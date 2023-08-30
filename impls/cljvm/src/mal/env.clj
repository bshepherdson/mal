(ns mal.env
  (:require
   [mal.schema :as ms]
   [mal.util.malli :as mu]))

(mu/defn nest-env :- ms/Env
  [outer :- [:maybe ms/Env]]
  {:contents (atom {})
   :parent   outer})

(def empty-env (nest-env nil))

(mu/defn env-search :- [:or [:= ::not-found] ms/Value]
  [{:keys [parent] :as env}  :- ms/Env
   sym                       :- :symbol
   not-found                 :- fn?]
  (let [contents @(:contents env)]
    (if (contains? contents sym)
      (get contents sym)
      (if parent
        (recur parent sym)
        (not-found sym)))))

(mu/defn env-find :- ms/Value
  [env :- ms/Env
   sym :- :symbol]
  (env-search env sym (constantly nil)))

(mu/defn env-get :- ms/Value
  [env :- ms/Env
   sym :- :symbol]
  (env-search
    env sym
    #(throw (ex-info (str "undefined symbol: " (name %))
                     {:mal.error/undefined-symbol %}))))

(mu/defn env-set :- ms/Env
  [env   :- ms/Env
   sym   :- :symbol
   value :- ms/Value]
  (swap! (:contents env) assoc sym value)
  env)
