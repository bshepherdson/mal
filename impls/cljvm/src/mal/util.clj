(ns mal.util)

(defn listy? [x]
  (or (list? x) (seq? x)))

(defn atom? [x]
  (instance? clojure.lang.Atom x))
