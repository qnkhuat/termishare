(ns colab.core-test
  (:require [clojure.test :refer :all]
            [termishare.core :refer :all]))

(defn meaning-of-life
  []
  {:answer 42})

(deftest meaning-of-life-test
  (testing "FIXME, I fail."
    (let [a (meaning-of-life)]
      (is (= {:answer 42} a)))))
