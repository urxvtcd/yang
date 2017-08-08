package device

import (
	"strings"

	"github.com/c2stack/c2g/val"

	"github.com/c2stack/c2g/meta"
	"github.com/c2stack/c2g/node"
	"github.com/c2stack/c2g/nodes"
)

type ProxyContextKey int

const RemoteIpAddressKey ProxyContextKey = 0

type DeviceAddresser func(id string, d Device) string

func MapNode(mgr *Map, addresser DeviceAddresser, onRegister ProtocolHandler) node.Node {
	return &nodes.Basic{
		OnChild: func(r node.ChildRequest) (node.Node, error) {
			switch r.Meta.GetIdent() {
			case "device":
				return deviceRecordListNode(mgr.devices, addresser), nil
			}
			return nil, nil
		},
		OnAction: func(r node.ActionRequest) (node.Node, error) {
			switch r.Meta.GetIdent() {
			case "register":
				var hnd DeviceHnd
				if err := r.Input.InsertInto(deviceHndNode(&hnd)).LastErr; err != nil {
					return nil, err
				}
				ctx := r.Selection.Context
				if regAddr, hasRegAddr := ctx.Value(RemoteIpAddressKey).(string); hasRegAddr {
					hnd.Address = strings.Replace(hnd.Address, "{REQUEST_ADDRESS}", regAddr, 1)
				}
				if d, err := onRegister(hnd.Address); err != nil {
					return nil, err
				} else {
					mgr.Add(hnd.DeviceId, d)
				}
				return nil, nil
			}
			return nil, nil
		},
		OnNotify: func(r node.NotifyRequest) (node.NotifyCloser, error) {
			switch r.Meta.GetIdent() {
			case "update":
				sub := mgr.OnUpdate(func(d Device, id string, c Change) {
					n := deviceChangeNode(id, d, addresser, c)
					r.Send(n)
				})
				return sub.Close, nil
			}
			return nil, nil
		},
	}
}

func deviceChangeNode(id string, d Device, addresser DeviceAddresser, c Change) node.Node {
	return &nodes.Extend{
		Node: deviceNode(id, d, addresser),
		OnField: func(p node.Node, r node.FieldRequest, hnd *node.ValueHandle) error {
			switch r.Meta.GetIdent() {
			case "change":
				var err error
				hnd.Val, err = node.NewValue(r.Meta.GetDataType(), int(c))
				if err != nil {
					return err
				}
			default:
				return p.Field(r, hnd)
			}
			return nil
		},
	}
}

func deviceRecordListNode(devices map[string]Device, addresser DeviceAddresser) node.Node {
	index := node.NewIndex(devices)
	return &nodes.Basic{
		OnNext: func(r node.ListRequest) (node.Node, []val.Value, error) {
			var d Device
			var id string
			key := r.Key
			if key != nil {
				id = key[0].String()
				d = devices[id]
			} else {
				if v := index.NextKey(r.Row); v != node.NO_VALUE {
					if id = v.String(); id != "" {
						if d = devices[id]; d != nil {
							key = []val.Value{val.String(id)}
						}
					}
				}
			}
			if d != nil {
				return deviceNode(id, d, addresser), key, nil
			}
			return nil, nil, nil
		},
	}
}

func deviceHndNode(hnd *DeviceHnd) node.Node {
	return nodes.Reflect(hnd)
}

func deviceNode(id string, d Device, addresser DeviceAddresser) node.Node {
	return &nodes.Basic{
		OnChild: func(r node.ChildRequest) (node.Node, error) {
			switch r.Meta.GetIdent() {
			case "module":
				return deviceModuleList(d.Modules()), nil
			}
			return nil, nil
		},
		OnField: func(r node.FieldRequest, hnd *node.ValueHandle) error {
			switch r.Meta.GetIdent() {
			case "deviceId":
				hnd.Val = val.String(id)
			case "address":
				hnd.Val = val.String(addresser(id, d))
			}
			return nil
		},
	}
}

func deviceModuleList(mods map[string]*meta.Module) node.Node {
	index := node.NewIndex(mods)
	return &nodes.Basic{
		OnNext: func(r node.ListRequest) (node.Node, []val.Value, error) {
			key := r.Key
			var m *meta.Module
			if r.Key != nil {
				m = mods[r.Key[0].String()]
			} else {
				if v := index.NextKey(r.Row); v != node.NO_VALUE {
					module := v.String()
					if m = mods[module]; m != nil {
						key = []val.Value{val.String(m.GetIdent())}
					}
				}
			}
			if m != nil {
				return deviceModuleNode(m), key, nil
			}
			return nil, nil, nil
		},
	}
}

func deviceModuleNode(m *meta.Module) node.Node {
	return &nodes.Basic{
		OnField: func(r node.FieldRequest, hnd *node.ValueHandle) error {
			switch r.Meta.GetIdent() {
			case "name":
				hnd.Val = val.String(m.GetIdent())
			case "revision":
				hnd.Val = val.String(m.Revision.GetIdent())
			}
			return nil
		},
	}
}

type RegistrationRequest struct {
	Address string
	Port    string
	Id      string
}

func registrationRequest(s node.Selection) (RegistrationRequest, error) {
	var reg RegistrationRequest
	if err := s.InsertInto(nodes.Reflect(&reg)).LastErr; err != nil {
		return reg, err
	}
	return reg, nil
}
