(ns mal.util
  (:refer-clojure :exclude [fn?]))

(defn listy? [x]
  (or (list? x) (seq? x)))

(defn atom? [x]
  (instance? clojure.lang.Atom x))

(defn fn? [obj]
  (boolean (or (clojure.core/fn? obj)
               (and (map? obj)
                    (:fn obj)
                    (not (:macro? obj))))))
