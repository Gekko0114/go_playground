package main

import (
	"fmt"
	"math/rand"
	"time"
)

var randGenerator = func(max float64) float64 {
	rand.Seed(time.Now().UnixNano())
	r := rand.Float64() * max
	return r
}

func weightedChoiceOne(v int, w []float64) float64 {
	vs := make([]int, 0, v)
	for i := 0; i < v; i++ {
		vs = append(vs, i)
	}

	var sum float64
	for _, v := range w {
		sum += v
	}
	r := randGenerator(sum)
	for j, v := range vs {
		r -= w[j]
		if r < 0 {
			return float64(v)
		}
	}
	return 0
}

func weightedChoice(v, size int, w []float64) ([]float64, error) {
	vs := make([]int, 0, v)
	for i := 0; i < v; i++ {
		vs = append(vs, i)
	}
	var sum float64
	for _, v := range w {
		sum += v
	}

	result := make([]float64, 0, size)
	for i := 0; i < size; i++ {
		r := randGenerator(sum)
		for j, v := range vs {
			r -= w[j]
			if r < 0 {
				result = append(result, float64(v))
				sum -= w[j]
				//選択された値を削除している。
				w = append(w[:j], w[j+1:]...)
				vs = append(vs[:j], vs[j+1:]...)

				break
			}
		}
	}
	return result, nil
}

func main() {
	r1 := weightedChoiceOne(5, []float64{0.1, 0.1, 0.2, 0.9, 0.1})
	r2, _ := weightedChoice(5, 4, []float64{0.1, 0.9, 0.2, 0.3, 0.1})
	fmt.Println(r1)
	fmt.Println(r2)
}
