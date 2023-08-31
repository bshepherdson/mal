(ns mal.env
  (:require
   [mal.schema :as ms]
   [mal.util.malli :as mu]))

(mu/defn env-search :- [:or [:= ::not-found] ::ms/value]
  [{:keys [parent] :as env}  :- ::ms/env
   sym                       :- :symbol
   not-found                 :- fn?]
  (let [contents @(:contents env)
        ;; Strip off clojure.core/ but not any other namespaces.
        sym      (if (and (qualified-symbol? sym)
                          (= (namespace sym) "clojure.core"))
                   (symbol (name sym))
                   sym)]
    (if (contains? contents sym)
      (get contents sym)
      (if parent
        (recur parent sym not-found)
        (not-found sym)))))

(mu/defn env-find :- ::ms/value
  [env :- ::ms/env
   sym :- :symbol]
  (env-search env sym (constantly nil)))

(mu/defn env-get :- ::ms/value
  [env :- ::ms/env
   sym :- :symbol]
  (env-search
    env sym
    #(throw (ex-info (str "undefined symbol: " (name %))
                     {:mal.error/undefined-symbol %
                      :mal/error (str "'" % "' not found")}))))

(mu/defn env-set :- ::ms/env
  [env   :- ::ms/env
   sym   :- :symbol
   value :- ::ms/value]
  (swap! (:contents env) assoc sym value)
  env)

(mu/defn nest-env :- ::ms/env
  ([outer :- [:maybe ::ms/env]]
   {:contents (atom {})
    :parent   outer})

  ([outer :- [:maybe ::ms/env]
    binds :- [:maybe [:sequential :symbol]]
    exprs :- [:maybe [:sequential ::ms/value]]]
   (let [[bind& bind-tail] (take-last 2 binds)]
     (if (= bind& '&)
       ;; Variadic args
       (let [fixed     (- (count binds) 2)
             fixed-env (nest-env outer (take fixed binds) (take fixed exprs))]
         (env-set fixed-env bind-tail (or (drop fixed exprs) ())))
       ;; Just fixed args
       (reduce (fn [e [b x]] (env-set e b x))
               (nest-env outer)
               (map vector binds exprs))))))

(def empty-env (nest-env nil))
