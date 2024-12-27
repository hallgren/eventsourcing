package main

import (
	"context"
	"fmt"
	"time"

	"github.com/hallgren/eventsourcing"
	"github.com/hallgren/eventsourcing/eventstore/memory"
	"github.com/hallgren/eventsourcing/example/order"
)

func main() {
	es := memory.Create()
	repo := eventsourcing.NewEventRepository(es)
	repo.Register(&order.Order{})

	orderMap := make(map[string]string)
	startedCount := 0
	completedCount := 0
	discountAppliedCount := 0
	moneyMade := 0

	go func() {

		i := 0
		for {
			o, err := order.Create(100)
			if err != nil {
				panic(err)
			}
			err = o.AddDiscount(10)
			if err != nil {
				panic(err)
			}
			err = o.Pay(80)
			if err != nil {
				panic(err)
			}
			// 50% of orders are not completed
			if i%2 == 0 {
				err = o.Pay(10)
				if err != nil {
					panic(err)
				}
			}
			err = repo.Save(o)
			if err != nil {
				panic(err)
			}
			time.Sleep(time.Second)
			i++
		}
	}()

	for {
		p := repo.Projections.Projection(es.All(0, 3), func(e eventsourcing.Event) error {
			if e.Reason() == "Created" {
				orderMap[e.AggregateID()] = "Created"
				startedCount++
			} else if e.Reason() == "Completed" {
				completedCount++
				delete(orderMap, e.AggregateID())
			} else if e.Reason() == "DiscountApplied" {
				discountAppliedCount++
			} else if e.Reason() == "Paid" {
				switch event := e.Data().(type) {
				case *order.Paid:
					moneyMade += int(event.Amount)
				}
			}

			fmt.Println(startedCount, completedCount, discountAppliedCount, moneyMade)
			return nil
		})
		p.Run(context.Background(), time.Second*2)
	}
}
