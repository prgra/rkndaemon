package daemon

import (
	"rkndelta/downloader"
	"rkndelta/resolver"
	"time"
)

type App struct {
	Downloader *downloader.Downloader
	Resolver   *resolver.Resolver
	Config     Config
}

type Config struct {
	KknURL         string
	DNSServers     []string
	WorkerCount    int
	ResolverFile   string
	SocialInterval int
	DumpInterval   int
}

func New(c Config) (a *App, err error) {
	dwn, err := downloader.New(c.KknURL)
	if err != nil {
		return a, err
	}
	res := resolver.New(c.DNSServers)
	res.Run(c.WorkerCount, c.ResolverFile)
	return &App{
		Downloader: dwn,
		Resolver:   res,
		Config:     c,
	}, nil
}

func (a *App) Run() {
	go a.Downloader.DumpDownloader(time.Duration(a.Config.DumpInterval) * time.Minute)
	a.Downloader.SocialDownloader(time.Duration(a.Config.SocialInterval) * time.Minute)
}
