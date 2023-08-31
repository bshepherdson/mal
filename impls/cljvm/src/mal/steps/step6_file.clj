(ns mal.steps.step6-file
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
    (let [[f a b c] ast]
      ;; Special forms, or the default eval for lists.
      (case f
        def! (let [value (mal-eval b env)]
               (env/env-set env a value)
               value)

        let* (let [let-env (reduce (fn [e [k v]]
                                     (env/env-set e k (mal-eval v e)))
                                   (env/nest-env env)
                                   (partition 2 a))]
               (recur b let-env)) ;; TCO

        do   (do (-> ast
                     rest
                     drop-last
                     list*
                     (eval-ast env))
                 (recur (last ast) env))

        if   (if (mal-eval a env)
               (recur b env)
               (recur c env))

        fn*  {:mal/type :fn
              :ast       b
              :env       env
              :params    a
              :fn        (fn [& args]
                           (mal-eval b (env/nest-env env a args)))}

        ;; Default case for applying functions.
        (let [[f & args] (eval-ast ast env)]
          (if (map? f)
            (recur (:ast f) (env/nest-env (:env f) (:params f) args))
            (apply f args)))))

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
          (:mal.error/undefined-symbol data)
          (str (:mal.error/undefined-symbol data) " not found")

          :else (throw e))))))

;; Mal-defined functions
(rep "(def! not (fn* (a) (if a false true)))")
(rep "(def! load-file (fn* (f) (eval (read-string (str \"(do \" (slurp f) \"\\nnil)\")))))")

(defn -main [& _]
  (loop []
    (print "user> ")
    (flush)
    (let [input (read-line)]
      (if (and input (pos? (count input)))
        (do
          (println (rep input))
          (recur))
        :eof))))

(comment
  (rep "(def! a (atom 7))")
  (rep "(def! inc3 (fn* [x] (+ x 3)))")
  (rep "(inc3 4)")
  (rep "a")
  (rep "(swap! a + 3)")
  (rep "(swap! a inc3)")
  *e
  )
