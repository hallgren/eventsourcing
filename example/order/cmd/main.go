package main

import (
	"context"
	"fmt"
	"time"

	"github.com/hallgren/eventsourcing"
	"github.com/hallgren/eventsourcing/aggregate"
	"github.com/hallgren/eventsourcing/eventstore/memory"
	"github.com/hallgren/eventsourcing/example/order"
)

type Order struct {
	DiscountAmount uint
	Total          uint
}

func main() {
	es := memory.Create()
	aggregate.Register(&order.Order{})

	ongoingOrders := make(map[string]*Order)
	completedCount := 0
	moneyMade := 0
	totalDiscountAmount := 0

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
			// 33% of orders remove discount
			if i%3 == 0 {
				o.RemoveDiscount()
			}
			err = o.Pay(80)
			if err != nil {
				panic(err)
			}
			// 25% of orders are not completed
			if i%4 == 0 {
				err = o.Pay(o.Outstanding)
				if err != nil {
					panic(err)
				}
			}
			err = aggregate.Save(es, o)
			if err != nil {
				panic(err)
			}
			time.Sleep(time.Second)
			i++
		}
	}()

	for {
		// setup how the projection will handle events and build the read model
		p := eventsourcing.NewProjection(es.All(0, 3), func(e eventsourcing.Event) error {
			switch event := e.Data().(type) {
			// When an order is created add it to an order map
			case *order.Created:
				ongoingOrders[e.AggregateID()] = &Order{
					Total: event.Total,
				}
			// When order is complete add the discount amount to a sum also store the total amount of completed orders
			case *order.Completed:
				completedCount++
				totalDiscountAmount += int(ongoingOrders[e.AggregateID()].DiscountAmount)
				// delete the order from the map
				delete(ongoingOrders, e.AggregateID())
			// Store the discount amount in the order in the map
			case *order.DiscountApplied:
				ongoingOrders[e.AggregateID()].DiscountAmount = ongoingOrders[e.AggregateID()].Total - event.Total
			// If the discount is removed reset the discount amount in the orders map
			case *order.DiscountRemoved:
				ongoingOrders[e.AggregateID()].DiscountAmount = 0
			// If a payment is made store the value
			case *order.Paid:
				moneyMade += int(event.Amount)
			}
			fmt.Println("active order:", len(ongoingOrders), "completed orders:", completedCount, "money made:", moneyMade, "total discount amount:", totalDiscountAmount)
			return nil
		})
		p.Run(context.Background(), time.Second*2)
	}
}
