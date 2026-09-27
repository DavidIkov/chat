package chats

import (
	"chat/src/shared"
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"
)

// JoinLink is an in-memory invite that lets other users join a chat. Join links
// are never persisted: they live only in the server's memory and disappear once
// the server stops.
type JoinLink struct {
	Token         string
	ChatUID       shared.UID
	ExpiresAt     shared.Time // 0 means the link lives until the server stops
	RemainingUses uint        // 0 means the link can be used unlimited times
}

type joinLinkStore struct {
	links map[string]JoinLink
}

func newJoinLinkStore() *joinLinkStore {
	return &joinLinkStore{links: make(map[string]JoinLink)}
}

func createJoinLinkToken() string {
	tokenBuff := make([]byte, 16)
	rand.Read(tokenBuff)
	return hex.EncodeToString(tokenBuff)
}

func (this *joinLinkStore) isExpired(link JoinLink) bool {
	return link.ExpiresAt != 0 && shared.Time(time.Now().UnixMilli()) >= link.ExpiresAt
}

func (this *joinLinkStore) removeExpired() {
	for token, link := range this.links {
		if this.isExpired(link) {
			delete(this.links, token)
		}
	}
}

func (this *joinLinkStore) removeChatLinks(chatUID shared.UID) {
	for token, link := range this.links {
		if link.ChatUID == chatUID {
			delete(this.links, token)
		}
	}
}

func (this *joinLinkStore) lookup(token string) (JoinLink, bool) {
	link, ok := this.links[token]
	if !ok {
		return JoinLink{}, false
	}
	if this.isExpired(link) {
		delete(this.links, token)
		return JoinLink{}, false
	}
	return link, true
}

func (this *joinLinkStore) consume(token string) {
	link, ok := this.links[token]
	if !ok || link.RemainingUses == 0 {
		return
	}
	link.RemainingUses--
	if link.RemainingUses == 0 {
		delete(this.links, token)
		return
	}
	this.links[token] = link
}

// CreateJoinLink creates a new invite for chatUID and returns it.
//
// lifetimeSeconds controls how long the link stays valid: 0 means it lives until
// the server stops. maxUses caps how many users can join with it: 0 means
// unlimited.
func (this *ChatsService) CreateJoinLink(chatUID shared.UID, lifetimeSeconds int64, maxUses uint) JoinLink {
	link := JoinLink{
		Token:         createJoinLinkToken(),
		ChatUID:       chatUID,
		RemainingUses: maxUses,
	}
	if lifetimeSeconds > 0 {
		link.ExpiresAt = shared.Time(time.Now().UnixMilli()) + shared.Time(lifetimeSeconds)*1000
	}

	this.mutex.Lock()
	defer this.mutex.Unlock()

	this.joinLinks.removeExpired()
	this.joinLinks.links[link.Token] = link

	return link
}

// JoinChatByLink uses the join link behind token to add userUID to its chat and
// returns the chat uid.
//
// Managing the link's uses is the store's concern, not the caller's: a use is
// spent only when a new membership is created (re-joining is a no-op that does
// not burn a use), and the link is removed once its last use is spent. It
// returns InvalidJoinLinkError when the token is unknown or expired.
func (this *ChatsService) JoinChatByLink(ctx context.Context, token string, userUID shared.UID) (shared.UID, error) {
	this.mutex.Lock()
	defer this.mutex.Unlock()

	link, ok := this.joinLinks.lookup(token)
	if !ok {
		return 0, InvalidJoinLinkError
	}

	joined, err := this.JoinChat(ctx, link.ChatUID, userUID)
	if err != nil {
		return 0, err
	}

	if joined {
		this.joinLinks.consume(token)
	}

	return link.ChatUID, nil
}

