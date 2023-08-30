(ns mal.steps.step1_read_print
  "Totally cheating and using built-in Clojure structures and Edamame to parse."
  (:require
   [clojure.walk :as walk]
   [edamame.core :as edamame]
   [mal.printer :as printer]
   [mal.schema :as ms]
   [mal.util.malli :as mu]))

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

(mu/defn mal-eval :- ms/Value
  [ast :- ms/Value]
  ast)

(mu/defn mal-print :- :string
  [value :- ms/Value]
  (printer/mal-pr-str value))

(mu/defn rep :- :string
  [input :- :string]
  (try
    (-> input
        mal-read
        mal-eval
        mal-print)
    (catch Throwable e
      (let [data (ex-data e)]
        (cond
          (:edamame/expected-delimiter data)
          (str "unexpected EOF, expected " (:edamame/expected-delimiter data))

          :else (str "unknown error: " data))))))

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
