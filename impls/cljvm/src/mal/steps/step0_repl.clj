(ns mal.steps.step0_repl
  (:require
   [mal.util.malli :as mu]))

(mu/defn mal-read :- :string
  [input :- :string]
  input)

(mu/defn mal-eval :- :string
  [ast :- :string]
  ast)

(mu/defn mal-print :- :string
  [value :- :string]
  value)

(mu/defn rep :- :string
  [input :- :string]
  (-> input
      mal-read
      mal-eval
      mal-print))

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
