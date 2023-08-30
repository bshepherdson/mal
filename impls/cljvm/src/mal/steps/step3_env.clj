(ns mal.steps.step3-env
  "Totally cheating and using built-in Clojure structures and Edamame to parse."
  (:require
   [clojure.walk :as walk]
   [edamame.core :as edamame]
   [mal.env :as env]
   [mal.printer :as printer]
   [mal.schema :as ms]
   [mal.util.malli :as mu]))

(def ^:private repl-env
  (-> env/empty-env
      (env/env-set '+ +)
      (env/env-set '- -)
      (env/env-set '* *)
      (env/env-set '/ quot)))

(defn- syntax-quote [form]
  ;; The form already has unquote and unquote-splicing in it.
  ;; Mal wants it to be called splice-unquote, so we can rename that.
  (letfn [(adjust [x]
            (if (and (symbol? x)
                     (#{'unquote-splicing 'clojure.core/unquote-splicing} x))
              'splice-unquote
              x))]
    (list 'quasiquote
          (if (sequential? form)
            (walk/postwalk adjust form)
            (adjust form)))))

(defn- postprocess [{:keys [obj loc]}]
  (letfn [(post [x]
            (if (and (qualified-symbol? x)
                     (= (namespace x) "clojure.core"))
              (symbol (name x))
              x))]
    (if (sequential? obj)
      (walk/postwalk post obj)
      obj)))

(def ^:private edamame-options
  {:deref        true
   :quote        true
   :syntax-quote syntax-quote
   :postprocess  postprocess})

(mu/defn mal-read :- ms/Value
  [input :- :string]
  (edamame/parse-string input edamame-options))

(declare mal-eval)

(mu/defn eval-ast :- ms/Value
  [ast :- ms/Value
   env :- ms/Env]
  (cond
    (symbol? ast) (env/env-get env ast)
    (list? ast)   (doall (map #(mal-eval % env) ast))
    (vector? ast) (mapv #(mal-eval % env) ast)
    (map? ast)    (update-vals ast #(mal-eval % env))
    :else         ast))

(mu/defn mal-eval :- ms/Value
  [ast :- ms/Value
   env :- ms/Env]
  (cond
    (and (list? ast)
         (= '() ast))
    '() ; Empty list special case

    (list? ast)
    (let [[f a b c] ast]
      (case f
        def! (let [value (mal-eval b env)]
               (env/env-set env a value)
               value)

        let* (let [let-env (reduce (fn [e [k v]]
                                     (env/env-set e k (mal-eval v e)))
                                   (env/nest-env env)
                                   (partition 2 a))]
               (mal-eval b let-env))

        (let [[f & args] (eval-ast ast env)]
          (apply f args))))

    :else (eval-ast ast env)))

(mu/defn mal-print :- :string
  [value :- ms/Value]
  (printer/mal-pr-str value))

(mu/defn rep :- :string
  [input :- :string]
  (try
    (-> input
        mal-read
        (mal-eval repl-env)
        mal-print)
    (catch Throwable e
      (let [data (ex-data e)]
        (cond
          (:edamame/expected-delimiter data)
          (str "unexpected EOF, expected " (:edamame/expected-delimiter data))
          (:mal.error/undefined-symbol data)
          (str (:mal.error/undefined-symbol data) " not found")

          :else (str "unknown error: " (or data (.printStackTrace e))))))))

(defn -main []
  (loop []
    (print "user> ")
    (flush)
    (let [input (read-line)]
      (if (and input (pos? (count input)))
        (do
          (println (rep input))
          (recur))
        :eof))))
