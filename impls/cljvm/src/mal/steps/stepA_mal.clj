(ns mal.steps.stepA-mal
  "Totally cheating and using built-in Clojure structures and Edamame to parse."
  (:require
   [mal.core :as core]
   [mal.env :as env]
   [mal.printer :as printer]
   [mal.reader :as reader]
   [mal.schema :as ms]
   [mal.util :as u]
   [mal.util.malli :as mu]))

(declare mal-eval)

(def ^:private repl-env
  (reduce (fn [e [sym val]] (env/env-set e sym val))
          env/empty-env
          (merge @core/core-ns
                 {'eval (fn [form]
                          (mal-eval form @#'repl-env))})))

(declare mal-quasiquote)

(defn- qq-list [[elt :as elts]]
  (cond
    (empty? elts)
    ()

    (and (u/listy? elt)
         (= 'splice-unquote (first elt)))
    (list 'concat (second elt) (qq-list (rest elts)))

    :else (list 'cons (mal-quasiquote elt) (qq-list (rest elts)))))

(defn- mal-quasiquote [ast]
  (cond
    (u/listy? ast)    (let [[f a] ast]
                        (if (= f 'unquote)
                          a
                          (qq-list ast)))
    (vector? ast)     (list 'vec (qq-list (seq ast)))
    (or (symbol? ast)
        (map? ast))   (list 'quote ast)
    :else             ast))

(defn- macro-call? [ast env]
  (and (u/listy? ast)
       (symbol? (first ast))
       (let [f (env/env-find env (first ast))]
         (and (map? f) (:macro? f)))))

(defn- mal-macroexpand [ast env]
  (if (macro-call? ast env)
    ;; We know it's a macro, so it's a Mal function.
    (let [f (env/env-get env (first ast))
          expanded (mal-eval (:ast f)
                             (env/nest-env (:env f) (:params f) (rest ast)))]
      (recur expanded env))
    ;; No longer a macro, just return ast.
    ast))

(mu/defn eval-ast :- ::ms/value
  [ast :- ::ms/value
   env :- ::ms/env]
  (cond
    (symbol? ast)  (env/env-get env ast)
    (u/listy? ast) (doall (map #(mal-eval % env) ast))
    (vector? ast)  (mapv #(mal-eval % env) ast)
    (map? ast)     (update-vals ast #(mal-eval % env))
    :else          ast))

(mu/defn mal-eval :- ::ms/value
  [ast :- ::ms/value
   env :- ::ms/env]
  (cond
    (and (u/listy? ast) (= '() ast))
    '() ; Empty list special case

    (u/listy? ast)
    (let [expanded (mal-macroexpand ast env)]
      (if (not (u/listy? expanded))
        ;; No longer a list after macro expansion.
        (eval-ast expanded env)

        ;; Special forms, or the default eval for lists.
        (let [[f a b c :as ast] expanded]
          (case f
            def!
            (let [value (mal-eval b env)
                  value (if (and (meta b)
                                 (instance? clojure.lang.IObj value))
                          (vary-meta value merge (meta b))
                          value)]
              (env/env-set env a value)
              value)

            defmacro!
            (let [value (mal-eval b env)
                  value (cond-> value
                          (and (map? value) (= (:mal/type value) :fn))
                          (assoc :macro? true))]
              (env/env-set env a value)
              value)

            macroexpand (mal-macroexpand a env)

            let*
            (let [let-env (reduce (fn [e [k v]]
                                    (env/env-set e k (mal-eval v e)))
                                  (env/nest-env env)
                                  (partition 2 a))]
              (recur b let-env)) ;; TCO

            quote            a
            quasiquoteexpand (mal-quasiquote a)
            quasiquote       (recur (mal-quasiquote a) env)

            do   (do (-> ast
                         rest
                         drop-last
                         list*
                         (eval-ast env))
                     (recur (last ast) env))

            if   (if (mal-eval a env)
                   (recur b env)
                   (recur c env))

            fn*  (-> {:mal/type :fn
                      :ast       b
                      :env       env
                      :params    a
                      :fn        (fn [& args]
                                   (mal-eval b (env/nest-env env a args)))}
                     (with-meta (meta ast)))

            try* (if-let [[catch-sym err-sym catch-body] (when (u/listy? b) b)]
                   ;; There's something listy in the b slot, it must be a catch* clause.
                   (do
                     (when (not= catch-sym 'catch*)
                       (throw (ex-info "not found: missing catch* clause" {:catch b})))
                     (try
                       (mal-eval a env)
                       (catch Exception err
                         ;; If we caught a host error rather than a Mal one, re-throw it.
                         (let [mal-err   (or (:mal/error (ex-data err))
                                             (ex-message err))]
                           (mal-eval catch-body (env/nest-env env [err-sym] [mal-err]))))))
                   ;; The b slot is either missing or not a catch* clause.
                   ;; Missing is OK, but anything else is no go.
                   (if (= (count ast) 2)
                     ;; Just (try* a), tail-recurse on it.
                     (recur a env)
                     (throw (ex-info "malformed catch* clause" {:found b}))))

            ;; Default case for applying functions.
            (let [[f & args] (eval-ast ast env)]
              (if (map? f)
                (recur (:ast f) (env/nest-env (:env f) (:params f) args))
                (apply f args)))))))

    :else (eval-ast ast env)))

(mu/defn mal-print :- :string
  [value :- ::ms/value]
  (binding [printer/*print-readably* true]
    (printer/mal-pr-str value)))

(mu/defn rep :- :string
  [input :- :string]
  (try
    (-> input
        reader/mal-read
        (mal-eval repl-env)
        mal-print)
    (catch Throwable e
      (let [data (ex-data e)]
        (cond
          (:edamame/expected-delimiter data)
          (str "unexpected EOF, expected " (:edamame/expected-delimiter data))

          (:mal/error data)
          (str "Error: " (binding [printer/*print-readably* false]
                           (printer/mal-pr-str (:mal/error data))))

          :else (throw e))))))

;; Mal-defined functions
(rep "(def! *host-language* \"cljvm\")")
(rep "(def! not (fn* (a) (if a false true)))")
(rep "(def! load-file (fn* (f) (eval (read-string (str \"(do \" (slurp f) \"\\nnil)\")))))")
(rep "(defmacro! cond (fn* (& xs) (if (> (count xs) 0) (list 'if (first xs) (if (> (count xs) 1) (nth xs 1) (throw \"odd number of forms to cond\")) (cons 'cond (rest (rest xs)))))))")

(defn- repl []
  (loop []
    (print "user> ")
    (flush)
    (let [input (read-line)]
      (cond
        (nil? input)   :eof
        (empty? input) (recur) ;; Silently loop on empty input
        :else          (do
                         (println (rep input))
                         (recur))))))

(defn -main [& _]
  (env/env-set repl-env '*ARGV* (rest *command-line-args*))
  (if-let [script (first *command-line-args*)]
    (rep (str "(load-file \"" script "\")"))
    (do
      (rep (str "(println (str \"Mal [\" *host-language* \"]\"))"))
      (repl))))
