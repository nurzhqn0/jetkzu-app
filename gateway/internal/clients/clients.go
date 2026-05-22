package clients

import (
	"fmt"

	driverv1 "github.com/jetkzu/jetkzu/gen/go/driver/v1"
	notifv1 "github.com/jetkzu/jetkzu/gen/go/notification/v1"
	paymentv1 "github.com/jetkzu/jetkzu/gen/go/payment/v1"
	ridev1 "github.com/jetkzu/jetkzu/gen/go/ride/v1"
	userv1 "github.com/jetkzu/jetkzu/gen/go/user/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Clients struct {
	User         userv1.UserServiceClient
	Driver       driverv1.DriverServiceClient
	Ride         ridev1.RideServiceClient
	Payment      paymentv1.PaymentServiceClient
	Notification notifv1.NotificationServiceClient

	conns []*grpc.ClientConn
}

func dial(addr string) (*grpc.ClientConn, error) {
	conn, err := grpc.NewClient(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultServiceConfig(`{"loadBalancingPolicy":"round_robin"}`),
	)
	if err != nil {
		return nil, fmt.Errorf("dial %s: %w", addr, err)
	}
	return conn, nil
}

func Connect(userAddr, driverAddr, rideAddr, paymentAddr, notifAddr string) (*Clients, error) {
	c := &Clients{}
	addrs := []string{userAddr, driverAddr, rideAddr, paymentAddr, notifAddr}
	conns := make([]*grpc.ClientConn, 0, len(addrs))
	for _, a := range addrs {
		conn, err := dial(a)
		if err != nil {
			for _, cc := range conns {
				_ = cc.Close()
			}
			return nil, err
		}
		conns = append(conns, conn)
	}
	c.conns = conns
	c.User = userv1.NewUserServiceClient(conns[0])
	c.Driver = driverv1.NewDriverServiceClient(conns[1])
	c.Ride = ridev1.NewRideServiceClient(conns[2])
	c.Payment = paymentv1.NewPaymentServiceClient(conns[3])
	c.Notification = notifv1.NewNotificationServiceClient(conns[4])
	return c, nil
}

func (c *Clients) Close() {
	for _, cc := range c.conns {
		_ = cc.Close()
	}
}
