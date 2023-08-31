(ns mal.schema
  (:require
   [malli.core :as mc]
   [malli.registry :as mr]
   [mal.util :as u]))

;; TODO: Fix the schema.
(def registry
  (merge (mc/default-schemas)
         {::bindings :any #_[:map-of :symbol [:ref ::value]]
          ::atom     :any #_[:fn  #(and (u/atom? %)
                                 (mc/validate [:ref ::bindings] (deref %)))]
          ::value    :any #_[:or {:error/message "Mal value"}
                      :int
                      :string
                      :symbol
                      :keyword
                      :boolean
                      :nil
                      fn?
                      [:ref ::atom]
                      [:map {:error/message "Mal function"}
                       [:mal/type [:= :fn]]
                       [:ast [:ref ::value]]
                       [:env [:ref ::env]]
                       [:params [:sequential :symbol]]
                       [:fn fn?]]
                      [:vector [:ref ::value]]
                      [:sequential [:ref ::value]]
                      [:map-of [:ref ::value] [:ref ::value]]]
          ::env      :any #_[:map
                      [:parent   [:maybe [:ref ::env]]]
                      [:contents [:ref ::atom]]]}))

(mr/set-default-registry! registry)
