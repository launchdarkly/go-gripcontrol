package pubcontrol

var (
	defaultSubscriptionWatcher SubscriptionWatcher = &alwaysTrueSubscriptionWatcher{}
)

// Tracks subscribed channels. For Fanout, this would utilize: https://fanout.io/docs/devguide.html#subscription-feeds
type SubscriptionWatcher interface {
	Start()
	Stop()
	// Checks if subscriptions are being updated without errors. Useful for scenarios where you want
	// to deliver all messages when subscription status is not available, or just general monitoring.
	IsHealthy() bool

	// Checks if a channel is connected.
	IsChannelConnected(channel string) bool
}

type alwaysTrueSubscriptionWatcher struct{}

func (w *alwaysTrueSubscriptionWatcher) Start()                                 {}
func (w *alwaysTrueSubscriptionWatcher) Stop()                                  {}
func (w *alwaysTrueSubscriptionWatcher) IsHealthy() bool                        { return true }
func (w *alwaysTrueSubscriptionWatcher) IsChannelConnected(channel string) bool { return true }
