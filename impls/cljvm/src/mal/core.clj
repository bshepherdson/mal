(ns mal.core
  "Core namespace of built-in functions."
  (:require
   [clojure.string :as str]
   [mal.printer :as printer]
   [mal.reader :as reader]
   [mal.util :as u]))

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

#_(defn- mal-wrapped
  {:style/indent 1}
  [sym f]
  (swap! core-ns assoc sym f)
  ['mal-wrapped sym])

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
(mal-def 'list?  u/listy?)
(mal-def 'empty? empty?)
(mal-def 'count  count)
(mal-def '=      =)
(mal-def '<      <)
(mal-def '<=     <=)
(mal-def '>      >)
(mal-def '>=     >=)

(mal-def 'cons   cons)
(mal-def 'concat concat)
(mal-def 'vec    vec)
(mal-def 'first  first)
(mal-def 'rest   rest)
(mal-def 'nth    nth)

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

(mal-def 'read-string reader/mal-read)
(mal-def 'slurp       slurp)

;; Atoms
(mal-def 'atom        atom)
(mal-def 'atom?       #(instance? clojure.lang.Atom %))
(mal-def 'deref       deref)
(mal-def 'reset!      reset!)
(mal-def 'swap!
         (fn [a f & args]
           (let [ff (if (map? f)
                      (:fn f)
                      f)]
             (apply swap! a ff args))))
