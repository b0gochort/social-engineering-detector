package scraper

import (
	"context"
	"log"

	"github.com/chromedp/chromedp"
)

// Scraper manages the browser and scraping tasks.
type Scraper struct {
	ctx    context.Context
	cancel context.CancelFunc
}

// NewScraper creates and initializes a new scraper instance.
func NewScraper(headless bool) (*Scraper, error) {
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", headless),
	)

	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)

	ctx, cancelCtx := chromedp.NewContext(allocCtx, chromedp.WithLogf(log.Printf))

	return &Scraper{
		ctx: ctx,
		cancel: func() {
			cancelCtx()
			cancel()
		},
	}, nil
}

// Close closes the browser and releases resources.
func (s *Scraper) Close() {
	s.cancel()
}

// Navigate opens a URL and waits for the page to stabilize.
func (s *Scraper) Navigate(url string) error {
	return chromedp.Run(s.ctx,
		chromedp.Navigate(url),
		// Wait for the chat list to appear, which means the page is loaded
		chromedp.WaitVisible(`//div[contains(@class, 'chatlist')]`),
	)
}
