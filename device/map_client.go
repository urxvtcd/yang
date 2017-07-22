package device

import (
	"github.com/c2stack/c2g/c2"
	"github.com/c2stack/c2g/node"
)

type MapClient struct {
	proto       ProtocolHandler
	browser     *node.Browser
	baseAddress string
}

func NewMapClient(d Device, baseAddress string, proto ProtocolHandler) *MapClient {
	b, err := d.Browser("map")
	if err != nil {
		panic(err)
	}
	return &MapClient{
		proto:       proto,
		browser:     b,
		baseAddress: baseAddress,
	}
}

type NotifySubscription node.NotifyCloser

func (self NotifySubscription) Close() error {
	return node.NotifyCloser(self)()
}

func (self *MapClient) Device(id string) (Device, error) {
	sel := self.browser.Root().Find("device=" + id)
	if sel.LastErr != nil {
		return nil, sel.LastErr
	}
	return self.device(sel)
}

type DeviceHnd struct {
	DeviceId string
	Address  string
}

func (self *MapClient) device(sel node.Selection) (Device, error) {
	var address string
	if v, err := sel.GetValue("address"); err != nil {
		return nil, err
	} else {
		address = v.Str
	}
	c2.Debug.Printf("map client address %s", self.baseAddress+address)
	return self.proto(self.baseAddress + address)
}

func (self *MapClient) OnUpdate(l ChangeListener) c2.Subscription {
	return self.onUpdate("update", l)
}

func (self *MapClient) OnModuleUpdate(module string, l ChangeListener) c2.Subscription {
	return self.onUpdate("update?filter=module/name%3d'"+module+"'", l)
}

func (self *MapClient) onUpdate(path string, l ChangeListener) c2.Subscription {
	closer, err := self.browser.Root().Find(path).Notifications(func(msg node.Selection) {
		id, err := msg.GetValue("deviceId")
		if err != nil {
			c2.Err.Print(err)
			return
		}
		d, err := self.device(msg)
		if err != nil {
			c2.Err.Print(err)
			return
		}
		l(d, id.Str, Added)
	})
	if err != nil {
		panic(err)
	}
	return NotifySubscription(closer)
}