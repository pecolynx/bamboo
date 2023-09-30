package bamboo

import (
	"errors"
)

type GoroutineBambooPubSubMap interface {
	CreateChannel(channelName string) chan []byte
	GetChannel(channelName string) (chan []byte, error)
	ClosePublishChannel(channelName string) error
	CloseSubscribeChannel(channelName string) error
}

type goroutineBambooPubSubMap struct {
	pubsubMap   map[string]chan []byte
	closePubMap map[string]chan interface{}
	closeSubMap map[string]chan interface{}
}

func NewGoroutineBambooPubSubMap() GoroutineBambooPubSubMap {
	return &goroutineBambooPubSubMap{
		pubsubMap:   make(map[string]chan []byte),
		closePubMap: make(map[string]chan interface{}),
		closeSubMap: make(map[string]chan interface{}),
	}
}

func (m *goroutineBambooPubSubMap) CreateChannel(channelName string) chan []byte {
	pubsub := make(chan []byte)
	m.pubsubMap[channelName] = pubsub

	closePub := make(chan interface{}, 1)
	m.closePubMap[channelName] = closePub

	closeSub := make(chan interface{}, 1)
	m.closeSubMap[channelName] = closeSub

	go func() {
		<-closePub
		<-closeSub
		close(pubsub)
		delete(m.pubsubMap, channelName)
	}()

	return pubsub
}

func (m *goroutineBambooPubSubMap) GetChannel(channelName string) (chan []byte, error) {
	if _, ok := m.pubsubMap[channelName]; !ok {
		return nil, errors.New("NotFOundxx")

	}
	pubsub := m.pubsubMap[channelName]

	return pubsub, nil
}

func (m *goroutineBambooPubSubMap) ClosePublishChannel(channelName string) error {
	if _, ok := m.closePubMap[channelName]; !ok {
		return errors.New("NotFound")
	}
	m.closePubMap[channelName] <- struct{}{}
	return nil
}

func (m *goroutineBambooPubSubMap) CloseSubscribeChannel(channelName string) error {
	if _, ok := m.closeSubMap[channelName]; !ok {
		return errors.New("NotFound")
	}
	closeSub := m.closeSubMap[channelName]
	closeSub <- struct{}{}
	return nil
}
