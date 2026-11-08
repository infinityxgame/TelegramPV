package telegram

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gotd/td/session"
	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/auth"
	"github.com/gotd/td/tg"

	"telegrampv/pkg/config"
	"telegrampv/pkg/i18n"
	"telegrampv/pkg/logbus"
	"telegrampv/pkg/storage"
)

type UserInfo struct {
	ID        int64  `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Username  string `json:"username"`
	Phone     string `json:"phone"`
	ColorID   *int   `json:"color_id,omitempty"`
}

type AuthStatus struct {
	Configured     bool      `json:"configured"`
	HasCredentials bool      `json:"has_credentials"`
	Authorized     bool      `json:"authorized"`
	Authenticated  bool      `json:"authenticated"`
	State          string    `json:"state"`
	Phone          string    `json:"phone,omitempty"`
	User           *UserInfo `json:"user,omitempty"`
}

type ClientManager struct {
	apiID               int
	apiHash             string
	client              *telegram.Client
	rawClient           *tg.Client
	sessionPath         string
	cancelRun           context.CancelFunc
	runWg               sync.WaitGroup
	mu                  sync.RWMutex
	readyChan           chan struct{}
	connErr             error
	phoneCodeHash       string
	pendingPhone        string
	channelAccessHashes map[int64]int64
	userAccessHashes    map[int64]int64
	chatNames           map[int64]string
	chatUsernames       map[int64]string
}

func NewClientManager() *ClientManager {
	config.InitPaths()
	sessPath := filepath.Join(config.DataDir, "tg_session.json")

	cm := &ClientManager{
		sessionPath:         sessPath,
		channelAccessHashes: make(map[int64]int64),
		userAccessHashes:    make(map[int64]int64),
		chatNames:           make(map[int64]string),
		chatUsernames:       make(map[int64]string),
	}
	return cm
}

func systemVersionLabel() string {
	switch runtime.GOOS {
	case "windows":
		return "Windows"
	case "darwin":
		return "macOS"
	case "linux":
		return "Linux"
	default:
		return runtime.GOOS
	}
}

func (cm *ClientManager) HaySesionGuardada() bool {
	if cm == nil {
		return false
	}
	cm.mu.RLock()
	sessionPath := cm.sessionPath
	cm.mu.RUnlock()

	_, err := os.Stat(sessionPath)
	return err == nil
}

func (cm *ClientManager) InitClient(apiIDStr, apiHash string) error {
	apiIDStr = strings.TrimSpace(apiIDStr)
	apiHash = strings.TrimSpace(apiHash)

	borrarSesion := apiIDStr == "" || apiHash == ""
	id := 0
	if !borrarSesion {
		var err error
		id, err = strconv.Atoi(apiIDStr)
		if err != nil {
			return fmt.Errorf(i18n.T("telegram.invalidAPIID"), err)
		}
	}

	cm.stopRunning()

	cm.mu.Lock()
	defer cm.mu.Unlock()

	if borrarSesion {
		cm.apiID = 0
		cm.apiHash = ""
		cm.client = nil
		cm.rawClient = nil
		return nil
	}

	cm.apiID = id
	cm.apiHash = apiHash
	cm.connErr = nil
	cm.readyChan = make(chan struct{})
	readyChan := cm.readyChan

	opts := telegram.Options{
		SessionStorage: &session.FileStorage{
			Path: cm.sessionPath,
		},
		Device: telegram.DeviceConfig{
			DeviceModel:   "TelegramPV Desktop",
			SystemVersion: systemVersionLabel(),
			AppVersion:    config.AppVersion,
			LangCode:      "es",
		},
	}

	client := telegram.NewClient(cm.apiID, cm.apiHash, opts)
	cm.client = client
	cm.rawClient = client.API()

	ctx, cancel := context.WithCancel(context.Background())
	cm.cancelRun = cancel

	readyOnce := sync.Once{}
	cm.runWg.Add(1)
	go func() {
		defer cm.runWg.Done()
		defer func() {
			readyOnce.Do(func() {
				close(readyChan)
			})
		}()

		logbus.Info(logbus.CatTelegram, i18n.T("telegram.connecting"))
		err := client.Run(ctx, func(runCtx context.Context) error {
			readyOnce.Do(func() {
				close(readyChan)
			})

			logbus.Success(logbus.CatTelegram, i18n.T("telegram.connected"))

			for {
				self, err := client.Self(runCtx)
				if err == nil && self != nil {
					logbus.Info(logbus.CatTelegram, i18n.T("telegram.sessionActive", self.FirstName, self.LastName, self.Username, self.ID))
					<-runCtx.Done()
					return runCtx.Err()
				}

				select {
				case <-runCtx.Done():
					return runCtx.Err()
				case <-time.After(3 * time.Second):
				}
			}
		})
		if err != nil && !errors.Is(err, context.Canceled) {
			logbus.Error(logbus.CatTelegram, i18n.T("telegram.runError", err))
			cm.mu.Lock()
			cm.connErr = err
			cm.mu.Unlock()
		}
	}()

	return nil
}

func (cm *ClientManager) stopRunning() {
	cm.mu.Lock()
	cancel := cm.cancelRun
	cm.cancelRun = nil
	cm.mu.Unlock()

	if cancel != nil {
		cancel()
	}

	done := make(chan struct{})
	go func() {
		cm.runWg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(1 * time.Second):
		logbus.Warn(logbus.CatTelegram, "Timeout esperando cierre de cliente anterior")
	}

	cm.mu.Lock()
	cm.client = nil
	cm.rawClient = nil
	cm.readyChan = nil
	cm.mu.Unlock()
}

func (cm *ClientManager) Stop() {
	cm.stopRunning()
}

func (cm *ClientManager) IsInitialized() bool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.client != nil && cm.apiID != 0 && cm.apiHash != ""
}

func (cm *ClientManager) WaitReady(ctx context.Context) error {
	cm.mu.RLock()
	ready := cm.readyChan
	client := cm.client
	connErr := cm.connErr
	cm.mu.RUnlock()

	if client == nil {
		return errors.New(i18n.T("telegram.notConfigured"))
	}
	if connErr != nil {
		return connErr
	}
	if ready == nil {
		return nil
	}

	select {
	case <-ready:
		cm.mu.RLock()
		err := cm.connErr
		cm.mu.RUnlock()
		return err
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(15 * time.Second):
		return errors.New(i18n.T("telegram.notConnected"))
	}
}

func (cm *ClientManager) GetAuthStatus(ctx context.Context) AuthStatus {
	cm.mu.RLock()
	client := cm.client
	apiID := cm.apiID
	apiHash := cm.apiHash
	phoneCodeHash := cm.phoneCodeHash
	pendingPhone := cm.pendingPhone
	cm.mu.RUnlock()

	if apiID == 0 || apiHash == "" || client == nil {
		return AuthStatus{
			Configured:     false,
			HasCredentials: false,
			Authorized:     false,
			Authenticated:  false,
			State:          "UNCONFIGURED",
		}
	}

	if !cm.HaySesionGuardada() {
		state := "NEED_PHONE"
		if phoneCodeHash != "" {
			state = "WAITING_CODE"
		}
		return AuthStatus{
			Configured:     true,
			HasCredentials: true,
			Authorized:     false,
			Authenticated:  false,
			State:          state,
			Phone:          pendingPhone,
		}
	}

	timeout := 5 * time.Second
	waitCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	if err := cm.WaitReady(waitCtx); err != nil {
		return AuthStatus{
			Configured:     true,
			HasCredentials: true,
			Authorized:     false,
			Authenticated:  false,
			State:          "NEED_PHONE",
		}
	}

	authClient := client.Auth()
	status, err := authClient.Status(ctx)
	if err != nil || !status.Authorized {
		return AuthStatus{
			Configured:     true,
			HasCredentials: true,
			Authorized:     false,
			Authenticated:  false,
			State:          "NEED_PHONE",
		}
	}

	user, err := client.Self(ctx)
	if err != nil {
		return AuthStatus{
			Configured:     true,
			HasCredentials: true,
			Authorized:     true,
			Authenticated:  true,
			State:          "LOGGED_IN",
		}
	}

	var colorID *int
	if pc, ok := user.Color.(*tg.PeerColor); ok {
		if c, ok := pc.GetColor(); ok {
			colorID = &c
		}
	}

	return AuthStatus{
		Configured:     true,
		HasCredentials: true,
		Authorized:     true,
		Authenticated:  true,
		State:          "LOGGED_IN",
		Phone:          user.Phone,
		User: &UserInfo{
			ID:        user.ID,
			FirstName: user.FirstName,
			LastName:  user.LastName,
			Username:  user.Username,
			Phone:     user.Phone,
			ColorID:   colorID,
		},
	}
}

func (cm *ClientManager) SendCode(ctx context.Context, phone string) (string, error) {
	cm.mu.Lock()
	client := cm.client
	cm.mu.Unlock()

	if client == nil {
		return "", errors.New(i18n.T("telegram.notConfigured"))
	}

	if err := cm.WaitReady(ctx); err != nil {
		return "", err
	}

	phone = strings.TrimSpace(phone)
	sentCode, err := client.Auth().SendCode(ctx, phone, auth.SendCodeOptions{})
	if err != nil {
		return "", fmt.Errorf(i18n.T("telegram.sendCodeError"), err)
	}

	sc, ok := sentCode.(*tg.AuthSentCode)
	if !ok {
		return "", errors.New(i18n.T("telegram.unexpectedResponse"))
	}

	cm.mu.Lock()
	cm.phoneCodeHash = sc.PhoneCodeHash
	cm.pendingPhone = phone
	cm.mu.Unlock()

	return sc.PhoneCodeHash, nil
}

func (cm *ClientManager) VerifyCode(ctx context.Context, phone, code, phoneCodeHash string) (string, error) {
	cm.mu.Lock()
	client := cm.client
	if phoneCodeHash == "" {
		phoneCodeHash = cm.phoneCodeHash
	}
	if phone == "" {
		phone = cm.pendingPhone
	}
	cm.mu.Unlock()

	if client == nil {
		return "", errors.New(i18n.T("telegram.notConfigured"))
	}

	if err := cm.WaitReady(ctx); err != nil {
		return "", err
	}

	_, err := client.Auth().SignIn(ctx, phone, code, phoneCodeHash)
	if err != nil {
		if errors.Is(err, auth.ErrPasswordAuthNeeded) || strings.Contains(strings.ToUpper(err.Error()), "SESSION_PASSWORD_NEEDED") {
			return "WAITING_2FA", nil
		}
		return "", fmt.Errorf(i18n.T("telegram.verifyCodeError"), err)
	}

	return "LOGGED_IN", nil
}

func (cm *ClientManager) TwoFactor(ctx context.Context, password string) error {
	cm.mu.Lock()
	client := cm.client
	cm.mu.Unlock()

	if client == nil {
		return errors.New(i18n.T("telegram.notConfigured"))
	}

	if err := cm.WaitReady(ctx); err != nil {
		return err
	}

	_, err := client.Auth().Password(ctx, password)
	if err != nil {
		return fmt.Errorf(i18n.T("telegram.2faError"), err)
	}

	return nil
}

func userDisplayName(u *tg.User) string {
	if u.Deleted {
		return "Deleted Account"
	}
	name := strings.TrimSpace(u.FirstName + " " + u.LastName)
	if name == "" {
		name = strings.TrimSpace(u.Username)
	}
	if name == "" {
		name = fmt.Sprintf("User %d", u.ID)
	}
	return name
}

func (cm *ClientManager) FetchDialogsAndSave(ctx context.Context, st *storage.Storage) ([]storage.Chat, error) {
	cm.mu.RLock()
	raw := cm.rawClient
	cm.mu.RUnlock()

	if raw == nil {
		return nil, errors.New(i18n.T("telegram.notConnectedShort"))
	}

	var offsetPeer tg.InputPeerClass = &tg.InputPeerEmpty{}
	var offsetID int = 0
	var offsetDate int = 0

	var allChats []storage.Chat
	seenIDs := make(map[int64]bool)

	for page := 0; page < 20; page++ {
		res, err := raw.MessagesGetDialogs(ctx, &tg.MessagesGetDialogsRequest{
			OffsetPeer: offsetPeer,
			OffsetID:   offsetID,
			OffsetDate: offsetDate,
			Limit:      100,
		})
		if err != nil {
			return nil, fmt.Errorf(i18n.T("telegram.fetchDialogsError"), err)
		}

		var dialogs []tg.DialogClass
		var chats []tg.ChatClass
		var users []tg.UserClass
		var messages []tg.MessageClass

		switch d := res.(type) {
		case *tg.MessagesDialogs:
			dialogs = d.Dialogs
			chats = d.Chats
			users = d.Users
			messages = d.Messages
		case *tg.MessagesDialogsSlice:
			dialogs = d.Dialogs
			chats = d.Chats
			users = d.Users
			messages = d.Messages
		case *tg.MessagesDialogsNotModified:
			break
		}

		if len(dialogs) == 0 && len(chats) == 0 {
			break
		}

		channelsMap := make(map[int64]*tg.Channel)
		chatsMap := make(map[int64]*tg.Chat)
		usersMap := make(map[int64]*tg.User)

		for _, c := range chats {
			switch ch := c.(type) {
			case *tg.Channel:
				channelsMap[ch.ID] = ch
				cm.mu.Lock()
				cm.channelAccessHashes[ch.ID] = ch.AccessHash
				cm.mu.Unlock()
			case *tg.Chat:
				chatsMap[ch.ID] = ch
			}
		}

		for _, u := range users {
			if usr, ok := u.(*tg.User); ok {
				usersMap[usr.ID] = usr
				cm.mu.Lock()
				cm.userAccessHashes[usr.ID] = usr.AccessHash
				cm.mu.Unlock()
			}
		}

		for _, dlg := range dialogs {
			d, ok := dlg.(*tg.Dialog)
			if !ok {
				continue
			}

			var chatItem storage.Chat

			switch p := d.Peer.(type) {
			case *tg.PeerChannel:
				if ch, exists := channelsMap[p.ChannelID]; exists {
					// Canal con prefijo canónico -100...
					canonicID, _ := strconv.ParseInt(fmt.Sprintf("-100%d", ch.ID), 10, 64)
					if seenIDs[canonicID] {
						continue
					}
					seenIDs[canonicID] = true

					chatItem.ID = canonicID
					chatItem.Title = ch.Title
					chatItem.Username = ch.Username
					chatItem.AccessHash = ch.AccessHash

					if ch.Broadcast {
						chatItem.Type = "channel"
					} else {
						chatItem.Type = "group"
					}
					allChats = append(allChats, chatItem)
				}
			case *tg.PeerChat:
				if ch, exists := chatsMap[p.ChatID]; exists {
					canonicID := -ch.ID
					if seenIDs[canonicID] {
						continue
					}
					seenIDs[canonicID] = true

					chatItem.ID = canonicID
					chatItem.Title = ch.Title
					chatItem.Type = "group"
					allChats = append(allChats, chatItem)
				}
			case *tg.PeerUser:
				if usr, exists := usersMap[p.UserID]; exists {
					if usr.Self {
						continue // Omitir Saved Messages
					}
					canonicID := usr.ID
					if seenIDs[canonicID] {
						continue
					}
					seenIDs[canonicID] = true

					chatItem.ID = canonicID
					chatItem.Title = userDisplayName(usr)
					chatItem.Username = usr.Username
					chatItem.AccessHash = usr.AccessHash

					if usr.Bot {
						chatItem.Type = "bot"
					} else {
						chatItem.Type = "user"
					}
					allChats = append(allChats, chatItem)
				}
			}
		}

		if len(messages) == 0 {
			break
		}

		lastMsg, ok := messages[len(messages)-1].(*tg.Message)
		if !ok {
			break
		}
		offsetID = lastMsg.ID
		offsetDate = lastMsg.Date

		switch p := lastMsg.PeerID.(type) {
		case *tg.PeerChannel:
			cm.mu.RLock()
			hash := cm.channelAccessHashes[p.ChannelID]
			cm.mu.RUnlock()
			offsetPeer = &tg.InputPeerChannel{ChannelID: p.ChannelID, AccessHash: hash}
		case *tg.PeerUser:
			cm.mu.RLock()
			hash := cm.userAccessHashes[p.UserID]
			cm.mu.RUnlock()
			offsetPeer = &tg.InputPeerUser{UserID: p.UserID, AccessHash: hash}
		case *tg.PeerChat:
			offsetPeer = &tg.InputPeerChat{ChatID: p.ChatID}
		default:
			offsetPeer = &tg.InputPeerEmpty{}
		}

		if len(dialogs) < 100 {
			break
		}
	}

	if len(allChats) > 0 {
		if err := st.SaveChats(allChats); err != nil {
			log.Printf("[TG CLIENT] Warning: Failed to save chats to storage: %v", err)
		}
	}

	return allChats, nil
}

func (cm *ClientManager) LeaveOrDeleteChat(ctx context.Context, st *storage.Storage, chatID int64) error {
	cm.mu.RLock()
	raw := cm.rawClient
	cm.mu.RUnlock()

	if raw == nil {
		return errors.New(i18n.T("telegram.notConnectedShort"))
	}

	// Obtener el chat de la base de datos para conocer su tipo y access_hash
	chats, err := st.GetChats("everything", strconv.FormatInt(chatID, 10))
	var chat *storage.Chat
	if err == nil {
		for _, c := range chats {
			if c.ID == chatID {
				chat = &c
				break
			}
		}
	}

	if chatID < 0 {
		idStr := strconv.FormatInt(chatID, 10)
		if strings.HasPrefix(idStr, "-100") {
			// Es un Canal o Supergrupo
			channelID, _ := strconv.ParseInt(strings.TrimPrefix(idStr, "-100"), 10, 64)
			var accessHash int64
			if chat != nil {
				accessHash = chat.AccessHash
			}
			if accessHash == 0 {
				cm.mu.RLock()
				accessHash = cm.channelAccessHashes[channelID]
				cm.mu.RUnlock()
			}

			inputChannel := &tg.InputChannel{
				ChannelID:  channelID,
				AccessHash: accessHash,
			}
			_, err := raw.ChannelsLeaveChannel(ctx, inputChannel)
			if err != nil {
				return fmt.Errorf(i18n.T("telegram.leaveChannelError"), err)
			}
		} else {
			// Grupo básico
			basicChatID := -chatID
			_, err := raw.MessagesDeleteChatUser(ctx, &tg.MessagesDeleteChatUserRequest{
				ChatID: basicChatID,
				UserID: &tg.InputUserSelf{},
			})
			if err != nil {
				return fmt.Errorf(i18n.T("telegram.leaveChannelError"), err)
			}
		}
	} else {
		// Es un Bot o Chat de Usuario Directo
		var accessHash int64
		if chat != nil {
			accessHash = chat.AccessHash
		}
		if accessHash == 0 {
			cm.mu.RLock()
			accessHash = cm.userAccessHashes[chatID]
			cm.mu.RUnlock()
		}

		inputPeer := &tg.InputPeerUser{
			UserID:     chatID,
			AccessHash: accessHash,
		}

		_, err := raw.MessagesDeleteHistory(ctx, &tg.MessagesDeleteHistoryRequest{
			JustClear: false,
			Revoke:    true,
			Peer:      inputPeer,
		})
		if err != nil {
			return fmt.Errorf(i18n.T("telegram.deleteHistoryError"), err)
		}
	}

	// Eliminar de la base de datos local
	_ = st.DeleteChat(chatID)
	return nil
}

func (cm *ClientManager) LeaveOrDeleteChatsBatch(ctx context.Context, st *storage.Storage, chatIDs []int64) map[int64]error {
	results := make(map[int64]error)
	for _, id := range chatIDs {
		err := cm.LeaveOrDeleteChat(ctx, st, id)
		results[id] = err
	}
	return results
}
