package helpers

import (
	"fmt"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

const TgBotToken = "your_bot's_token_goes_here"
const welcomeMessage = "Hello\n\r\n\r" +
	"I am @DrDelphi 's unofficial BoN monitor BOT and I am here to help you keep an eye on your nodes."
const addNodeMsg = "Node's public key ? (64 or 256 bytes)"
const broadcastMsg = "Reply with the message you want to broadcast"

var TgBot *tgbotapi.BotAPI

func ReplyReceived(message *tgbotapi.Message) {
	if message.ReplyToMessage.Text == addNodeMsg {
		dbUser, err := GetUserByTelegramID(uint32(message.From.ID))
		if err != nil {
			fmt.Printf("Unknown user TG ID = %v trying to add node %s\n\r", message.From.ID, message.Text)
			return
		}
		key := message.Text
		if len(key) != 64 && len(key) != 256 {
			msg := tgbotapi.NewMessage(int64(message.From.ID), "⛔️ Invalid key length")
			TgBot.Send(msg)
			return
		}
		var nKey, bKey string
		for _, a := range NetworkConfig.InitialNodes {
			if a.PubKey == key || a.Address == key {
				bKey = a.Address
				nKey = a.PubKey
			}
		}
		if bKey == "" || nKey == "" {
			msg := tgbotapi.NewMessage(int64(message.From.ID), "⛔️ Not a Validator key")
			TgBot.Send(msg)
			return
		}
		if (*dbUser).Nodes != nil {
			for _, n := range *dbUser.Nodes {
				if n.BalancesKey == bKey {
					msg := tgbotapi.NewMessage(int64(message.From.ID), "⛔️ You have already added this key")
					TgBot.Send(msg)
					return
				}
			}
		}
		node := NodeType{
			UserID:      dbUser.ID,
			BalancesKey: bKey,
			NodesKey:    nKey,
		}
		err = DB.AddNode(&node)
		if err == nil {
			*dbUser.Nodes = append(*dbUser.Nodes, &node)
			msg := tgbotapi.NewMessage(int64(message.From.ID), "✅ Node added")
			TgBot.Send(msg)
		}
	}
	if message.ReplyToMessage.Text == broadcastMsg {
		for _, u := range Users {
			if !u.IsAdmin {
				msg := tgbotapi.NewMessage(int64(u.TelegramID), fmt.Sprintf("📢 <b>Important message from DrDelphi</b>\n\r\n\r%s", message.Text))
				msg.ParseMode = "HTML"
				TgBot.Send(msg)
			}
		}
	}
	mainMenu(message.From)
}

func CallbackQueryReceived(cb *tgbotapi.CallbackQuery) {
	if cb.Data == "NodesStats" {
		TgBot.AnswerCallbackQuery(tgbotapi.NewCallback(cb.ID, "Nodes Stats"))
		dbUser, err := GetUserByTelegramID(uint32(cb.From.ID))
		if err == nil {
			sendNodesStats(cb.From.ID, dbUser)
		}
	}
	if cb.Data == "AddNode" {
		TgBot.AnswerCallbackQuery(tgbotapi.NewCallback(cb.ID, "Add node"))
		msg := tgbotapi.NewMessage(int64(cb.From.ID), addNodeMsg)
		msg.ReplyMarkup = tgbotapi.ForceReply{
			ForceReply: true,
			Selective:  false,
		}
		TgBot.Send(msg)
		return
	}
	if cb.Data == "Download" {
		TgBot.AnswerCallbackQuery(tgbotapi.NewCallback(cb.ID, "Download. Please Wait !"))
		fileable := tgbotapi.NewDocumentUpload(int64(cb.From.ID), "node.zip")
		TgBot.Send(fileable)
	}
	if cb.Data == "Broadcast" {
		TgBot.AnswerCallbackQuery(tgbotapi.NewCallback(cb.ID, "Broadcast message"))
		msg := tgbotapi.NewMessage(int64(cb.From.ID), broadcastMsg)
		msg.ReplyMarkup = tgbotapi.ForceReply{
			ForceReply: true,
			Selective:  false,
		}
		TgBot.Send(msg)
		return
	}
	if cb.Data == "LeaderBoard" {
		TgBot.AnswerCallbackQuery(tgbotapi.NewCallback(cb.ID, "Leader Board"))
		sendLeaderBoard(cb.From.ID)
	}
	if cb.Data[0] == ':' {
		TgBot.AnswerCallbackQuery(tgbotapi.NewCallback(cb.ID, "Ok"))
		var strID, text string
		if strings.HasPrefix(cb.Data, ":NodeDetails") {
			strID = strings.TrimPrefix(cb.Data, ":NodeDetails_")
			nodeID, _ := strconv.ParseUint(strID, 10, 32)
			sendNodeDetails(cb.From.ID, uint32(nodeID))
		}
		if strings.HasPrefix(cb.Data, ":RemoveNode") {
			strID = strings.TrimPrefix(cb.Data, ":RemoveNode_")
			nodeID, _ := strconv.ParseUint(strID, 10, 32)
			text = "✅ Node removed"
			err := DB.RemoveNode(uint32(nodeID))
			if err != nil {
				text = fmt.Sprintf("⛔️ %s", err)
			}
			msg := tgbotapi.NewMessage(int64(cb.From.ID), text)
			TgBot.Send(msg)
		}
	}
	mainMenu(cb.From)
}

func sendNodeDetails(tgID int, nodeID uint32) {
	for _, u := range Users {
		for _, n := range *u.Nodes {
			if n.ID == nodeID {
				shard := fmt.Sprintf("%v", n.Hb.ReceivedShard)
				if n.Hb.ReceivedShard == 4294967295 {
					shard = "meta"
				}
				str := fmt.Sprintf("`Name: %s\n\rVersion: %s\n\rIs Validator: %v\n\rShard:%s\n\r"+
					"BalancesKey: %s\n\rNodesKey: %s\n\rUpTime: %v\n\rDownTime: %v`",
					n.Hb.NodeDisplayName, n.Hb.VersionNumber, n.Hb.IsValidator, shard,
					n.BalancesKey, n.NodesKey, n.Hb.TotalUpTimeSec, n.Hb.TotalDownTimeSec)
				msg := tgbotapi.NewMessage(int64(tgID), str)
				msg.ParseMode = "markdown"
				TgBot.Send(msg)
				return
			}
		}
	}
	msg := tgbotapi.NewMessage(int64(tgID), "⛔️ Node not found")
	TgBot.Send(msg)
}

func CommandReceived(update tgbotapi.Update) {
	var dbUser *UserType
	var err error
	dbUser, err = GetUserByTelegramID(uint32(update.Message.From.ID))
	if err == nil {
		if dbUser.UserName != update.Message.From.UserName ||
			dbUser.FirstName != update.Message.From.FirstName ||
			dbUser.LastName != update.Message.From.LastName {
			dbUser.UserName = update.Message.From.UserName
			dbUser.FirstName = update.Message.From.FirstName
			dbUser.LastName = update.Message.From.LastName
			err := DB.UpdateUser(dbUser)
			if err != nil {
				ReportToAdmins(fmt.Sprintf("⛔️ An error occured while updating user ID %v in DB: %s", dbUser.ID, err))
			}
		}
	} else {
		ReportToAdmins(fmt.Sprintf("🧝‍♂️ New registered user: '%v' '%s' '%s' '%s'\n\r",
			update.Message.From.ID, update.Message.From.UserName, update.Message.From.FirstName, update.Message.From.LastName))
		dbUser = &UserType{
			TelegramID: uint32(update.Message.From.ID),
			UserName:   update.Message.From.UserName,
			FirstName:  update.Message.From.FirstName,
			LastName:   update.Message.From.LastName,
			IsAdmin:    false,
		}
		err = DB.AddUser(dbUser)
		if err != nil {
			ReportToAdmins("⛔️ An error occured while saving him in DB")
			return
		}
		Users = append(Users, dbUser)
	}
	if update.Message.Command() == "start" { // && update.Message.Chat.IsPrivate() {
		startCommandReceived(update.Message.From)
	}
}

func startCommandReceived(tgUser *tgbotapi.User) {
	msg := tgbotapi.NewMessage(int64(tgUser.ID), welcomeMessage)
	msg.ParseMode = "HTML"
	TgBot.Send(msg)
	mainMenu(tgUser)
}

func ReportToAdmins(message string) {
	for _, u := range Users {
		if u.IsAdmin {
			msg := tgbotapi.NewMessage(int64(u.TelegramID), message)
			TgBot.Send(msg)
		}
	}
}

func mainMenu(tgUser *tgbotapi.User) {
	dbUser, _ := GetUserByTelegramID(uint32(tgUser.ID))
	if dbUser == nil {
		return
	}
	if dbUser.LastMenuID > 0 {
		TgBot.DeleteMessage(tgbotapi.DeleteMessageConfig{
			ChatID:    int64(tgUser.ID),
			MessageID: dbUser.LastMenuID,
		})
	}
	var keyboard tgbotapi.InlineKeyboardMarkup
	msg := tgbotapi.NewMessage(int64(tgUser.ID), "How may I assist you ?")
	if dbUser.IsAdmin {
		keyboard = tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("📊 Nodes Stats", "NodesStats"),
				tgbotapi.NewInlineKeyboardButtonData("➕ Add node", "AddNode"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🏆 Leader Board", "LeaderBoard"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("📥 Download last version", "Download"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("📣 Broadcast message", "Broadcast"),
			),
		)
	} else {
		keyboard = tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("📊 Nodes Stats", "NodesStats"),
				tgbotapi.NewInlineKeyboardButtonData("➕ Add node", "AddNode"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🏆 Leader Board", "LeaderBoard"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("📥 Download last version", "Download"),
			),
		)
	}
	msg.ReplyMarkup = keyboard
	resp, _ := TgBot.Send(msg)
	dbUser.LastMenuID = resp.MessageID
}

func sendNodesStats(tgID int, dbUser *UserType) {
	if len(*dbUser.Nodes) == 0 {
		msg := tgbotapi.NewMessage(int64(tgID), "⛔️ No nodes added yet")
		TgBot.Send(msg)
		return
	}
	for i, v := range *dbUser.Nodes {
		var status, shard string
		if v.Hb.IsActive {
			status = "Online ✔️"
		} else {
			status = "Offline ⭕️"
		}
		if v.Hb.ReceivedShard == 4294967295 {
			shard = "meta"
		} else {
			shard = strconv.FormatUint(uint64(v.Hb.ReceivedShard), 10)
		}
		str := fmt.Sprintf("`Node %v/%v - %s\n\r\n\r"+
			"Name: %s\n\r"+
			"Shard: %s\n\r"+
			"UpTime: %v`", i+1, len(*dbUser.Nodes), status, v.Hb.NodeDisplayName, shard, v.Hb.TotalUpTimeSec)
		var keyboard = tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("📖 Details", fmt.Sprintf(":NodeDetails_%v", v.ID)),
				tgbotapi.NewInlineKeyboardButtonData("➖ Remove", fmt.Sprintf(":RemoveNode_%v", v.ID)),
			),
		)
		msg := tgbotapi.NewMessage(int64(tgID), str)
		msg.ParseMode = "markdown"
		msg.ReplyMarkup = keyboard
		TgBot.Send(msg)
	}
}

func sendLeaderBoard(tgID int) {
	top := make([]int, len(NetworkConfig.InitialNodes))
	for i, _ := range NetworkConfig.InitialNodes {
		top[i] = i
	}
	w := true
	idx := 0
	var tmp int
	for {
		var sum1, val1, sum2, val2 float64 = 0, 0, 0, 0
		sum1 = float64(NetworkConfig.InitialNodes[top[idx]].Hb.TotalUpTimeSec) + float64(NetworkConfig.InitialNodes[top[idx]].Hb.TotalDownTimeSec)
		if sum1 != 0 {
			val1 = float64(NetworkConfig.InitialNodes[top[idx]].Hb.TotalUpTimeSec) / sum1
		}
		sum2 = float64(NetworkConfig.InitialNodes[top[idx+1]].Hb.TotalUpTimeSec) + float64(NetworkConfig.InitialNodes[top[idx+1]].Hb.TotalDownTimeSec)
		if sum2 != 0 {
			val2 = float64(NetworkConfig.InitialNodes[top[idx+1]].Hb.TotalUpTimeSec) / sum2
		}
		if val1 < val2 {
			tmp = top[idx]
			top[idx] = top[idx+1]
			top[idx+1] = tmp
			w = false
		}
		idx++
		if idx >= len(top)-2 {
			if w {
				break
			} else {
				w = true
				idx = 0
			}
		}
	}
	msg := tgbotapi.NewMessage(int64(tgID), "<b>🏆 Leader Board</b>")
	msg.ParseMode = "HTML"
	TgBot.Send(msg)
	for i := 0; i < 5; i++ {
		shard := fmt.Sprintf("%v", NetworkConfig.InitialNodes[top[i]].Hb.ReceivedShard)
		if NetworkConfig.InitialNodes[top[i]].Hb.ReceivedShard == 4294967295 {
			shard = "meta"
		}
		str := fmt.Sprintf("`#%v %s - shard %s\n\r%s\n\rUp:%v Down:%v`", i+1,
			NetworkConfig.InitialNodes[top[i]].Hb.NodeDisplayName,
			shard,
			NetworkConfig.InitialNodes[top[i]].Address,
			NetworkConfig.InitialNodes[top[i]].Hb.TotalUpTimeSec,
			NetworkConfig.InitialNodes[top[i]].Hb.TotalDownTimeSec)
		msg := tgbotapi.NewMessage(int64(tgID), str)
		msg.ParseMode = "markdown"
		TgBot.Send(msg)
	}
	dbUser, err := GetUserByTelegramID(uint32(tgID))
	if err != nil {
		return
	}
	for i := 5; i < len(top); i++ {
		for _, n := range *dbUser.Nodes {
			if NetworkConfig.InitialNodes[top[i]].Address == n.BalancesKey {
				shard := fmt.Sprintf("%v", NetworkConfig.InitialNodes[top[i]].Hb.ReceivedShard)
				if NetworkConfig.InitialNodes[top[i]].Hb.ReceivedShard == 4294967295 {
					shard = "meta"
				}
				str := fmt.Sprintf("`#%v %s - shard %s\n\r%s\n\rUp:%v Down:%v`", i+1,
					NetworkConfig.InitialNodes[top[i]].Hb.NodeDisplayName,
					shard,
					NetworkConfig.InitialNodes[top[i]].Address,
					NetworkConfig.InitialNodes[top[i]].Hb.TotalUpTimeSec,
					NetworkConfig.InitialNodes[top[i]].Hb.TotalDownTimeSec)
				msg := tgbotapi.NewMessage(int64(tgID), str)
				msg.ParseMode = "markdown"
				TgBot.Send(msg)
			}
		}
	}
}
