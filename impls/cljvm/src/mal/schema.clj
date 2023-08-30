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
                                [:vector [:ref ::value]]
                                [:sequential [:ref ::value]]
                                [:map-of [:ref ::value] [:ref ::value]]]}}
   ::value])

(def Env
  [:schema {:registry {::env [:map
                              [:parent   [:maybe [:ref ::env]]]
                              [:contents [:map-of :symbol Value]]]}}
   ::env])
