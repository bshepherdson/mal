(ns mal.schema
  (:require
   [malli.core :as mc]))

(def Value
  [:schema {:registry {::value [:or {:error/message "Mal value"}
                                :int
                                :string
                                :symbol
                                :keyword
                                :boolean
                                :nil
                                fn?
                                [:map {:error/message "Mal function"}
                                 [:mal/type [:= :fn]]
                                 [:ast [:ref ::value]]
                                 [:env :any]
                                 [:params [:sequential :symbol]]
                                 [:fn fn?]]
                                [:vector [:ref ::value]]
                                [:sequential [:ref ::value]]
                                [:map-of [:ref ::value] [:ref ::value]]]}}
   ::value])

(def Bindings
  [:map-of :symbol Value])

(def Env
  [:schema {:registry {::env [:map
                              [:parent   [:maybe [:ref ::env]]]
                              [:contents [:fn #(and (instance? clojure.lang.Atom %)
                                                    (mc/validate Bindings (deref %)))]]]}}
   ::env])
