(ns build
  (:require [clojure.tools.build.api :as b]
            [clojure.java.shell :refer [sh]]))

(def basis (b/create-basis {:project "deps.edn"}))
(def src-dir "src/server/")
(def uber-file "target/colab.jar")
(def class-dir "target/classes")

(defn release-frontend
  []
  (let [install (sh "npm" "install")
        release (sh "npm" "run" "release")]
    (println "----------------- Install -----------------")
    (println (:out install))
    (println (:err install))
    (println "----------------- Release -----------------")
    (println (:out release))
    (println (:err release))))

(defn clean []
  (b/delete {:path "target"}))

(defn uber [_]
  (clean)
  (release-frontend)
  ;; Is not required but doesn't hurt anything
  (b/copy-dir {:src-dirs   [src-dir "resources"]
               :target-dir class-dir})
  (b/compile-clj {:basis     basis
                  :src-dirs  [src-dir]
                  :class-dir class-dir})
  (b/uber {:class-dir class-dir
           :uber-file uber-file
           :basis     basis
           :main      'server.core}))
