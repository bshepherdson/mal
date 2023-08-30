(ns mal.core
  "Core namespace of built-in functions."
  (:require
   [clojure.string :as str]
   [mal.printer :as printer]))

(def core-ns (atom {}))

#_(defmacro ^:private mal-def
  {:style/indent 0}
  [sym value]
  `(do (swap! core-ns assoc '~sym ~value)
       '~sym))

#_(defmacro ^:private mal-defn
  {:style/indent [:form]}
  [sym args & body]
  `(mal-def '~sym (fn ~args ~@body)))

(defn- mal-def
  {:style/indent 1}
  [sym value]
  (swap! core-ns assoc sym value)
  ['mal-def sym])

(mal-def '+ +)
(mal-def '- -)
(mal-def '* *)
(mal-def '/ quot)

(mal-def 'prn
         (fn [x]
           (binding [printer/*print-readably* true]
             (printer/mal-pr-str x))))

(mal-def 'list   list)
(mal-def 'list?  (fn [x] (or (list? x) (seq? x))))
(mal-def 'empty? empty?)
(mal-def 'count  count)
(mal-def '=      =)
(mal-def '<      <)
(mal-def '<=     <=)
(mal-def '>      >)
(mal-def '>=     >=)

(defn- mal-pr-str [& args]
  (binding [printer/*print-readably* true]
    (->> args
         (map printer/mal-pr-str)
         (str/join " "))))

(defn- mal-str [& args]
  (binding [printer/*print-readably* false]
    (->> args
         (map printer/mal-pr-str)
         (str/join ""))))

(defn- mal-println- [& args]
  (binding [printer/*print-readably* false]
    (->> args
         (map printer/mal-pr-str)
         (str/join " "))))

(mal-def 'pr-str mal-pr-str)
(mal-def 'str    mal-str)

(mal-def 'prn
         (fn [& args]
           (println (apply mal-pr-str args))
           nil))

(mal-def 'println
         (fn [& args]
           (println (apply mal-println- args))
           nil))
