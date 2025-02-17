package internal

import "fmt"

const ShortTerm = 50
const LongTerm = 200

type Service struct {
	client  Client
	store   Store
	last50  []float64
	last200 []float64
}

func NewService(client Client, store Store) *Service {
	return &Service{
		store:   store,
		client:  client,
		last50:  make([]float64, 0),
		last200: make([]float64, 0),
	}
}

func (s *Service) ManageRatesData(cValue float64, sValue string) {
	s.last50 = append(s.last50, cValue)
	if len(s.last50) > ShortTerm {
		s.last50 = s.last50[1:]
	}

	s.last200 = append(s.last200, cValue)
	if len(s.last200) > LongTerm {
		s.last200 = s.last200[1:]
	}

	if len(s.last50) == ShortTerm && len(s.last200) == LongTerm {
		if calculateAverage(s.last50) > calculateAverage(s.last200) {
			result, err := s.client.SellFunc(sValue)
			if err == nil {
				err = s.store.CreateOrderLog(result)
				if err != nil {
					fmt.Printf("Error sell order created log: %v\n", result)
				}
			}
		} else {
			result, err := s.client.BuyFunc(sValue)
			if err == nil {
				err = s.store.CreateOrderLog(result)
				if err != nil {
					fmt.Printf("Error buy order created log: %v\n", result)
				}
			}
		}
	}
}

func calculateAverage(data []float64) float64 {
	if len(data) == 0 {
		return 0
	}

	var sum float64

	for _, v := range data {
		sum += v
	}

	return sum / float64(len(data))
}
