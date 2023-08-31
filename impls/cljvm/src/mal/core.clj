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
(mal-def 'conj   conj)
(mal-def 'concat concat)
(mal-def 'vec    vec)
(mal-def 'first  first)
(mal-def 'rest   rest)
(mal-def 'nth
         (fn [xs i]
           (try
             (nth xs i)
             (catch IndexOutOfBoundsException _
               (throw (ex-info "index out of bounds" {:xs xs
                                                      :i  i
                                                      :mal/error "index out of bounds"
                                                      :mal.error/index-out-of-bounds i}))))))

(mal-def 'apply
         (fn [f & tail]
           (when (empty? tail)
             (throw (ex-info "apply requires at least 2 arguments" {})))
           (let [args (concat (drop-last tail) (last tail))
                 ff   (if (map? f) (:fn f) f)]
             (apply ff args))))

(mal-def 'map
         (fn [f xs]
           (let [ff (if (map? f) (:fn f) f)]
             (doall (map ff xs)))))

;; Type predicates
(mal-def 'fn?         u/fn?)
(mal-def 'map?        map?)
(mal-def 'nil?        nil?)
(mal-def 'true?       true?)
(mal-def 'false?      false?)
(mal-def 'macro?      #(boolean (and (map? %)
                                     (:fn %)
                                     (:macro? %))))
(mal-def 'number?     number?)
(mal-def 'string?     string?)
(mal-def 'symbol?     symbol?)
(mal-def 'vector?     vector?)
(mal-def 'keyword?    keyword?)
(mal-def 'sequential? sequential?)

;; Type conversion and construction
(mal-def 'symbol     symbol)
(mal-def 'keyword    keyword)
(mal-def 'vector     vector)
(mal-def 'hash-map   hash-map)
(mal-def 'seq        (fn [x]
                       (if (string? x)
                         (not-empty (map str x))
                         (seq x))))

;; Map functions
(mal-def 'assoc      assoc)
(mal-def 'dissoc     dissoc)
(mal-def 'get        get)
(mal-def 'contains?  contains?)
(mal-def 'keys       (fn [m] (or (keys m) ())))
(mal-def 'vals       (fn [m] (or (vals m) ())))

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
(mal-def 'readline    (fn [prompt]
                        (print prompt)
                        (flush)
                        (read-line)))

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

;; Exceptions
;; This wraps a Mal value into `{:mal/error ...}` on an `ex-info`.
(mal-def 'throw
         (fn [value]
           (throw (ex-info "(Mal internal throw)" {:mal/error value}))))

;; Metadata
(mal-def 'meta      #(-> % meta :mal/meta))
(mal-def 'with-meta (fn [obj metadata]
                      (vary-meta obj assoc :mal/meta metadata)))

(mal-def 'time-ms (fn [] (.toEpochMilli (java.time.Instant/now))))
