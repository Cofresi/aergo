package enterprise

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/aergoio/aergo/state"
	"github.com/aergoio/aergo/types"
)

type EnterpriseContext struct {
	Call   *types.CallInfo
	Args   []string
	Admins [][]byte
	Old    *Conf
}

func (e *EnterpriseContext) IsAdminExist(addr []byte) bool {
	for _, a := range e.Admins {
		if bytes.Equal(a, addr) {
			return true
		}
	}
	return false
}

func (e *EnterpriseContext) IsOldConfValue(value string) bool {
	if e.Old != nil {
		for _, v := range e.Old.Values {
			if v == value {
				return true
			}
		}
	}
	return false
}

func ExecuteEnterpriseTx(scs *state.ContractState, txBody *types.TxBody,
	sender *state.V) ([]*types.Event, error) {
	context, err := ValidateEnterpriseTx(txBody, sender, scs)
	if err != nil {
		return nil, err
	}
	var events []*types.Event
	switch context.Call.Name {
	case AppendAdmin:
		requestAddress := types.ToAddress(context.Args[0])
		err := setAdmins(scs,
			append(context.Admins, requestAddress))
		if err != nil {
			return nil, err
		}
	case RemoveAdmin:
		for i, v := range context.Admins {
			if bytes.Equal(v, types.ToAddress(context.Call.Args[0].(string))) {
				context.Admins = append(context.Admins[:i], context.Admins[i+1:]...)
				break
			}
		}
		err := setAdmins(scs, context.Admins)
		if err != nil {
			return nil, err
		}
	case SetConf:
		key := []byte(context.Args[0])
		confValues := context.Args[1:]
		err := setConfValues(scs, key, &Conf{
			Values: confValues,
		})
		if err != nil {
			return nil, err
		}
		events, err = createSetEvent(txBody.Recipient, string(key), confValues)
		if err != nil {
			return nil, err
		}
	case AppendConf:
		key := []byte(strings.ToUpper(context.Args[0]))
		appendValues := context.Args[1:]
		conf, err := getConf(scs, key)
		if err != nil {
			return nil, err
		}
		if conf == nil {
			conf = &Conf{On: false}
		}
		conf.Values = append(conf.Values, appendValues...)
		err = setConf(scs, key, conf)
		if err != nil {
			return nil, err
		}
		events, err = createSetEvent(txBody.Recipient, string(key), conf.Values)
		if err != nil {
			return nil, err
		}
	case RemoveConf:
		key := []byte(context.Args[0])
		removeValues := context.Args[1:]
		conf, err := getConf(scs, key)
		if err != nil {
			return nil, err
		}
		for _, v := range removeValues {
			conf.RemoveValue(v)
		}
		err = setConf(scs, key, conf)
		if err != nil {
			return nil, err
		}
		events, err = createSetEvent(txBody.Recipient, string(key), conf.Values)
		if err != nil {
			return nil, err
		}
	case EnableConf:
		key := []byte(context.Args[0])
		if err := enableConf(scs, key, context.Call.Args[1].(bool)); err != nil {
			return nil, err
		}
		jsonArgs, err := json.Marshal(context.Call.Args[1])
		if err != nil {
			return nil, err
		}
		events = append(events, &types.Event{
			ContractAddress: txBody.Recipient,
			EventName:       "Enable " + string(key),
			EventIdx:        0,
			JsonArgs:        string(jsonArgs),
		})
	default:
		return nil, fmt.Errorf("unsupported call in enterprise contract")
	}
	return events, nil
}

func createSetEvent(addr []byte, name string, v []string) ([]*types.Event, error) {
	jsonArgs, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return []*types.Event{
		&types.Event{
			ContractAddress: addr,
			EventName:       "Set " + name,
			EventIdx:        0,
			JsonArgs:        string(jsonArgs),
		},
	}, nil
}