package tools

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type DuckDuckGoSearchResult struct {
	Output   string                 `json:"output"`
	Metadata map[string]interface{} `json:"metadata"`
}

type resultItem struct {
	Title     string
	Link      string
	Snippet   string
	Thumbnail string
}

// WebSearch realiza una búsqueda web utilizando DuckDuckGo
//
// Parámetros:
//   - query: La consulta de búsqueda (requerido)
//   - max_results: Número máximo de resultados (opcional, por defecto 5, máximo 10)
//
// Ejemplo de uso:
//
//	result, err := WebSearch(map[string]interface{}{
//	    "query": "ejemplo de búsqueda",
//	    "max_results": 3,
//	})
func WebSearch(payload map[string]interface{}) (interface{}, error) {
	// Parsear parámetros
	query, ok := payload["query"].(string)
	if !ok || query == "" {
		return nil, &ToolError{Message: "el parámetro 'query' es requerido"}
	}

	maxResults := 5
	if mr, ok := payload["max_results"].(float64); ok {
		maxResults = int(mr)
	}
	if maxResults <= 0 || maxResults > 10 {
		maxResults = 5
	}

	// Crear la URL de búsqueda
	searchURL := fmt.Sprintf("https://html.duckduckgo.com/html/?q=%s", url.QueryEscape(query))

	// Crear cliente HTTP con timeout
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// Crear la petición
	req, err := http.NewRequest("GET", searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("error al crear la petición: %v", err)
	}

	// Añadir headers para parecer un navegador
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")

	// Realizar la petición
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error al realizar la petición: %v", err)
	}
	defer resp.Body.Close()

	// Parsear el HTML de la respuesta
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error al parsear el HTML: %v", err)
	}

	// Extraer resultados de búsqueda
	var results []resultItem

	doc.Find(".result").Each(func(i int, s *goquery.Selection) {
		if i >= maxResults {
			return
		}

		title := strings.TrimSpace(s.Find(".result__title").Text())
		link, _ := s.Find(".result__title a").Attr("href")
		snippet := strings.TrimSpace(s.Find(".result__snippet").Text())
		// Extraer imagen de vista previa si está disponible
		var thumbnailURL string
		if img := s.Find(".result__image img"); img.Length() > 0 {
			if src, exists := img.Attr("src"); exists {
				if strings.HasPrefix(src, "//") {
					src = "https:" + src
				}
				thumbnailURL = src
			}
		}

		if title != "" && link != "" {
			item := resultItem{
				Title:     title,
				Link:      link,
				Snippet:   snippet,
				Thumbnail: thumbnailURL,
			}
			results = append(results, item)
		}
	})

	// Si no se encontraron resultados, devolver un mensaje
	if len(results) == 0 {
		results = append(results, resultItem{
			Title:     "No se encontraron resultados",
			Link:      "",
			Snippet:   "No se encontraron resultados para la búsqueda: " + query,
			Thumbnail: "",
		})
	}

	// Formatear resultados
	var output strings.Builder
	metadata := make(map[string]interface{})

	// Metadatos generales
	metadata["query"] = query
	metadata["timestamp"] = time.Now().Format(time.RFC3339)
	metadata["result_count"] = len(results)
	metadata["source"] = "duckduckgo"

	// Estructura para almacenar los resultados en formato de lista
	var resultsList []map[string]interface{}

	for i, result := range results {
		// Crear un mapa para el resultado actual
		resultMap := map[string]interface{}{
			"position":    i + 1,
			"title":       result.Title,
			"url":         result.Link,
			"description": result.Snippet,
		}

		// Agregar la URL de la imagen de vista previa si está disponible
		if result.Thumbnail != "" {
			resultMap["thumbnail"] = result.Thumbnail
		}

		// Formatear el resultado actual
		if thumb, ok := resultMap["thumbnail"]; ok && thumb != "" {
			output.WriteString(fmt.Sprintf("%d. ![%s](%s) [%s](%s)\n", 
				i+1, result.Title, thumb, result.Title, result.Link))
		} else {
			output.WriteString(fmt.Sprintf("%d. [%s](%s)\n", i+1, result.Title, result.Link))
		}
		output.WriteString(fmt.Sprintf("   %s\n\n", result.Snippet))

		// Agregar a la lista de resultados
		resultsList = append(resultsList, resultMap)

		// Si ya tenemos suficientes resultados, salir
		if i >= maxResults-1 {
			break
		}
	}

	// Agregar la lista de resultados a los metadatos
	metadata["results"] = resultsList

	return &DuckDuckGoSearchResult{
		Output:   output.String(),
		Metadata: metadata,
	}, nil
}
