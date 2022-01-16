(ns build
  (:require [clojure.tools.build.api :as b]
            [clojure.java.shell :refer [sh]]))

(def class-dir "target/classes")
(def src-dir "src/server/")
(def basis (b/create-basis {:project "deps.edn"}))
(def uber-file "target/colab.jar")

(defn release-frontend
  []
  (sh "npm" "install")
  (sh "npm" "run" "release"))

(defn clean [_]
  (b/delete {:path "target"}))

(defn uber [_]
  (clean nil)

  (release-frontend)
  (b/copy-dir {:src-dirs [src-dir "resources"]
               :target-dir class-dir})

  (b/compile-clj {:basis basis
                  :src-dirs [src-dir]
                  :class-dir class-dir})

  (b/uber {:class-dir class-dir
           :uber-file uber-file
           :basis basis
           :main 'server.core}))
