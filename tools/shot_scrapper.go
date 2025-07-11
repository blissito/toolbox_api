package tools

import (
	"context"
	"time"

	"github.com/chromedp/chromedp"
)

// ShotScrapper toma una captura de pantalla de la URL dada y devuelve el buffer de la imagen PNG.
func ShotScrapper(url string) ([]byte, error) {
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	// Timeout para evitar bloqueos largos
	ctx, cancel = context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	var buf []byte
	err := chromedp.Run(ctx,
		chromedp.Navigate(url),
		chromedp.Sleep(2*time.Second), // Espera para cargar la página
		chromedp.FullScreenshot(&buf, 90),
	)
	if err != nil {
		return nil, err
	}
	return buf, nil
} 