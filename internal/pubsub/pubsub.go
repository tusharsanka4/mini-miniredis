package pubsub

import (
	"sort"
	"strings"
	"sync"

	"github.com/tusharsanka4/mini-miniredis/internal/resp"
)

// Send queues a complete reply without waiting for network I/O.
// It returns false when the client is closed or its outgoing queue is full.
type Subscriber interface {
	Send(reply string) bool
}

type Registry struct {
	mu       sync.Mutex
	channels map[string]map[Subscriber]bool
}

func New() *Registry {
	return &Registry{channels: make(map[string]map[Subscriber]bool)}
}

func (r *Registry) Subscribe(client Subscriber, channels []string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var replies []string
	for _, channel := range channels {
		if r.channels[channel] == nil {
			r.channels[channel] = make(map[Subscriber]bool)
		}
		r.channels[channel][client] = true
		replies = append(replies, ack("subscribe", resp.BulkString(channel), r.count(client)))
	}
	// Queue all acknowledgements together before a publisher can queue a message.
	// A single SUBSCRIBE with many channels must not fill the outgoing queue.
	if !client.Send(strings.Join(replies, "")) {
		r.remove(client)
	}
}

func (r *Registry) Unsubscribe(client Subscriber, channels []string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(channels) == 0 {
		for channel, subscribers := range r.channels {
			if subscribers[client] {
				channels = append(channels, channel)
			}
		}
		sort.Strings(channels)
	}
	if len(channels) == 0 {
		client.Send(ack("unsubscribe", resp.NullBulkString(), 0))
		return
	}
	var replies []string
	for _, channel := range channels {
		delete(r.channels[channel], client)
		if len(r.channels[channel]) == 0 {
			delete(r.channels, channel)
		}
		replies = append(replies, ack("unsubscribe", resp.BulkString(channel), r.count(client)))
	}
	if !client.Send(strings.Join(replies, "")) {
		r.remove(client)
	}
}

func (r *Registry) Publish(channel, message string) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	reply := resp.Array([]string{"message", channel, message})
	delivered := 0
	for client := range r.channels[channel] {
		if client.Send(reply) {
			delivered++
		} else {
			delete(r.channels[channel], client)
		}
	}
	if len(r.channels[channel]) == 0 {
		delete(r.channels, channel)
	}
	return delivered
}

func (r *Registry) Count(client Subscriber) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.count(client)
}

func (r *Registry) Remove(client Subscriber) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.remove(client)
}

// count and remove are called only while holding r.mu.
func (r *Registry) count(client Subscriber) int {
	count := 0
	for _, subscribers := range r.channels {
		if subscribers[client] {
			count++
		}
	}
	return count
}

func (r *Registry) remove(client Subscriber) {
	for channel, subscribers := range r.channels {
		delete(subscribers, client)
		if len(subscribers) == 0 {
			delete(r.channels, channel)
		}
	}
}

func ack(kind, channelReply string, count int) string {
	return resp.Replies([]string{resp.BulkString(kind), channelReply, resp.Integer(count)})
}
