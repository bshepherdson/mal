(ns mal.printer
  (:require
   [clojure.string :as str]
   [mal.util :as u]
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


(defn- mal-dispatch [x]
  (cond
    (and (map? x)
         (:mal/type x)) (:mal/type x)
    (map? x)            :map
    (vector? x)         :vector
    (u/listy? x)        :list
    (string? x)         :string
    (nil? x)            :nil
    (boolean? x)        :bool
    (fn? x)             :fn
    (number? x)         :number
    (symbol? x)         :symbol
    (keyword? x)        :keyword
    (u/atom? x)         :atom
    :else (throw (ex-info (str "mal-dispatch not defined for: " x) {:x x}))))

(defmulti mal-pr-str mal-dispatch)

(defmethod mal-pr-str :default [x]
  (str x))

(defmethod mal-pr-str :map [x]
  (wrap-seq "{" "}" (mapcat identity x)))

(defmethod mal-pr-str :vector [x]
  (wrap-seq "[" "]" x))

(defmethod mal-pr-str :list [x]
  (wrap-seq "(" ")" x))

(defmethod mal-pr-str :atom [x]
  (str "(atom " (mal-pr-str @x) ")"))

(defmethod mal-pr-str :string [x]
  (if *print-readably*
    (pr-str x)
    x))

(defmethod mal-pr-str :nil [_]
  ;; (str nil) ;=> ""
  "nil")

(defmethod mal-pr-str :fn [_]
  "#<function>")
