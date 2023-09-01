(ns mal.reader
  (:require
   [clojure.walk :as walk]
   [edamame.core :as edamame]
   [mal.schema :as ms]
   [mal.util :as u]
   [mal.util.malli :as mu]))

(defn- syntax-quote [form]
  ;; The form already has unquote and unquote-splicing in it.
  ;; Mal wants it to be called splice-unquote, so we can rename that.
  ;; TODO: That can be renamed to match Clojure once I'm done with Mal tests.
  (letfn [(adjust [x]
            (if (and (symbol? x)
                     (#{'unquote-splicing 'clojure.core/unquote-splicing} x))
              'splice-unquote
              x))]
    (list 'quasiquote
          (if (sequential? form)
            (walk/postwalk adjust form)
            (adjust form)))))

(defn- postprocess [{:keys [obj _loc]}]
  (let [symbold      (if (and (u/listy? obj)
                              (qualified-symbol? (first obj))
                              (= (namespace (first obj))
                                 "clojure.core"))
                       (-> obj first name symbol (cons (rest obj)))
                       obj)
        splice-fixed (if (and (u/listy? symbold)
                              (= (first symbold) 'unquote-splicing))
                       (cons 'splice-unquote (rest symbold))
                       symbold)]
    (if (meta obj)
      (list 'with-meta splice-fixed (meta obj))
      splice-fixed)))

(def ^:private edamame-options
  {:deref        true
   :quote        true
   :syntax-quote syntax-quote
   :postprocess  postprocess})

(mu/defn mal-read :- ::ms/value
  [input :- :string]
  (edamame/parse-string input edamame-options))
