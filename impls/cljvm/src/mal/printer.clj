(ns mal.printer
  (:require
   [clojure.string :as str]
   [mal.schema :as ms]
   [mal.util.malli :as mu]))

(defn- spaced [parts]
  (str/join " " parts))

(defn- wrap [left right middle]
  (str left middle right))

(declare mal-pr-str)

(defn- wrap-seq [left right coll]
  (->> coll
       (map mal-pr-str)
       spaced
       (wrap left right)))

(def ^:dynamic *print-readably* true)

(mu/defn mal-pr-str :- :string
  [value :- ms/Value]
  (cond
    (map? value)    (wrap-seq "{" "}" (mapcat identity value))
    (vector? value) (wrap-seq "[" "]" value)
    (list? value)   (wrap-seq "(" ")" value)
    (string? value) (if *print-readably*
                      (pr-str value)
                      value)
    (nil? value)    "nil"
    (fn? value)     "#<function>"
    :else           (str value)))
