package helpers

import (
	"fmt"
	"encoding/json"
	"io/ioutil"
	"net/http"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

type addressType struct {
	PubKey  string `json:"pubKey"`
	Address string `json:"address"`
	Hb 		heartBeatType
}

type nodesSetupType struct {
	StartTime                   uint64        `json:"startTime"`
	RoundDuration               int           `json:"roundDuration"`
	ConsensusGroupSize          int           `json:"consensusGroupSize"`
	MinNodesPerShard            int           `json:"minNodesPerShard"`
	MetaChainActive             bool          `json:"metaChainActive"`
	MetaChainConsensusGroupSize int           `json:"metaChainConsensusGroupSize"`
	MetaChainMinNodes           int           `json:"metaChainMinNodes"`
	InitialNodes                []addressType `json:"initialNodes"`
}

type heartBeatMessage struct {
	Message []heartBeatType `json:"message"`
}

type heartBeatType struct {
	HexPublicKey     string `json:"hexPublicKey"`
	TimeStamp        string `json:"timeStamp"`
	MaxInactive      string `json:"maxInactive"`
	IsActive         bool   `json:"isActive"`
	ReceivedShard    uint32 `json:"receivedShardID"`
	ComputedShard    uint32 `json:"computedShardID"`
	TotalUpTimeSec   uint32 `json:"totalUpTimeSec"`
	TotalDownTimeSec uint32 `json:"totalDownTimeSec"`
	VersionNumber    string `json:"versionNumber"`
	IsValidator      bool   `json:"isValidator"`
	NodeDisplayName  string `json:"nodeDisplayName"`
}

type balanceMessage struct {
	Balance uint64 `json:"balance"`
}

type NodeType struct {
	ID			uint32
	UserID		uint32
	BalancesKey	string
	NodesKey    string
	Hb			heartBeatType
}

const heartbeatHost = "https://wallet-api.elrond.com"

var NetworkConfig nodesSetupType

func LoadNodesJson() {
	req, err := http.NewRequest(
		http.MethodGet, "https://raw.githubusercontent.com/ElrondNetwork/elrond-config/master/nodesSetup.json", nil)
	if err != nil {
		panic("Unable to download nodesSetup.json from GitHub")
	}
	client := http.DefaultClient
	resp, err := client.Do(req)
	if err != nil {
		panic("Unable to download nodesSetup.json from GitHub")
	}
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic("Unable to download nodesSetup.json from GitHub")
	}
	json.Unmarshal(data, &NetworkConfig)
	resp.Body.Close()
	fmt.Printf("Loaded %v keys from nodesSetup.json\n\r", len(NetworkConfig.InitialNodes))
}

func GetHeartBeat() {
	var change bool
	firstTime := true
	for {
		req, err := http.NewRequest(http.MethodGet, heartbeatHost+"/node/heartbeatstatus", nil)
		if err != nil {
			continue
		}
		client := http.DefaultClient
		resp, err := client.Do(req)
		if err != nil {
			continue
		}
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			continue
		}
		var heartBeat heartBeatMessage
		json.Unmarshal(body, &heartBeat)
		resp.Body.Close()
		for _, hb := range heartBeat.Message {
			for _, u := range Users {
				for _, n := range *u.Nodes {
					if n.NodesKey != hb.HexPublicKey {
						continue
					}
					change = n.Hb.IsActive != hb.IsActive
					n.Hb = hb
					if change {
						var str string
						if hb.IsActive {
							str = fmt.Sprintf("✔️ Node online - %s", n.Hb.NodeDisplayName)
						} else {
							str = fmt.Sprintf("⭕ Node offline - %s", n.Hb.NodeDisplayName)
						}
						msg := tgbotapi.NewMessage(int64(u.TelegramID), str)
						if !firstTime {
							TgBot.Send(msg)
						}
					}
				}
			}
			for i, _ := range NetworkConfig.InitialNodes {
				a := &(NetworkConfig.InitialNodes[i])
				if a.PubKey == hb.HexPublicKey {
					a.Hb = hb
				}
			}
		}
		firstTime = false
	}
}
