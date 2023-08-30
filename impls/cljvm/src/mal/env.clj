(ns mal.env
  (:require
   [mal.schema :as ms]
   [mal.util.malli :as mu]))

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

(mu/defn nest-env :- ms/Env
  ([outer :- [:maybe ms/Env]]
   {:contents (atom {})
    :parent   outer})

  ([outer :- [:maybe ms/Env]
    binds :- [:maybe [:sequential :symbol]]
    exprs :- [:maybe [:sequential ms/Value]]]
   (let [[bind& bind-tail] (take-last 2 binds)]
     #_(prn "nest-env" binds bind& bind-tail)
     (if (= bind& '&)
       ;; Variadic args
       (let [fixed     (- (count binds) 2)
             fixed-env (nest-env outer (take fixed binds) (take fixed exprs))]
         #_(prn "variadic!" fixed (-> fixed-env :contents deref) bind-tail (drop fixed exprs))
         (env-set fixed-env bind-tail (or (drop fixed exprs) ())))
       ;; Just fixed args
       (reduce (fn [e [b x]] (env-set e b x))
               (nest-env outer)
               (map vector binds exprs))))))

(def empty-env (nest-env nil))
