package app

import (
	"github.com/planetary-social/scuttlego/service/app/commands"
	"github.com/planetary-social/scuttlego/service/app/queries"
)

type Application struct {
	Commands Commands
	Queries  Queries
}

type Commands struct {
	RedeemInvite         *commands.RedeemInviteHandler
	Follow               *commands.FollowHandler
	PublishRaw           *commands.PublishRawHandler
	PublishRawAsIdentity *commands.PublishRawAsIdentityHandler
	DownloadFeed         *commands.DownloadFeedHandler

	Connect       *commands.ConnectHandler
	DisconnectAll *commands.DisconnectAllHandler

	DownloadBlob *commands.DownloadBlobHandler
	CreateBlob   *commands.CreateBlobHandler

	AddToBanList      *commands.AddToBanListHandler
	RemoveFromBanList *commands.RemoveFromBanListHandler
	SetBanList        *commands.SetBanListHandler

	RoomsAliasRegister *commands.RoomsAliasRegisterHandler
	RoomsAliasRevoke   *commands.RoomsAliasRevokeHandler

	RunMigrations *commands.RunMigrationsHandler
}

type Queries struct {
	CreateHistoryStream  *queries.CreateHistoryStreamHandler
	ReceiveLog           *queries.ReceiveLogHandler
	PublishedLog         *queries.PublishedLogHandler
	Status               *queries.StatusHandler
	GetBlob              *queries.GetBlobHandler
	BlobDownloadedEvents *queries.BlobDownloadedEventsHandler
	RoomsListAliases     *queries.RoomsListAliasesHandler
	GetMessage           *queries.GetMessageHandler
	GetMessageBySequence *queries.GetMessageBySequenceHandler
}
