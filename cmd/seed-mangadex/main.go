package main

import (
	"MangaHub/pkg/models"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type apiResponse struct {
	Data   []mangaItem `json:"data"`
	Limit  int         `json:"limit"`
	Offset int         `json:"offset"`
	Total  int         `json:"total"`
}

type mangaItem struct {
	ID            string              `json:"id"`
	Attributes    mangaAttributes     `json:"attributes"`
	Relationships []mangaRelationship `json:"relationships"`
}

type mangaAttributes struct {
	Title       map[string]string `json:"title"`
	Description map[string]string `json:"description"`
	Status      string            `json:"status"`
	LastChapter string            `json:"lastChapter"`
	Tags        []mangaTag        `json:"tags"`
}

type mangaTag struct {
	Attributes tagAttributes `json:"attributes"`
}

type tagAttributes struct {
	Name  map[string]string `json:"name"`
	Group string            `json:"group"`
}

type mangaRelationship struct {
	ID         string                  `json:"id"`
	Type       string                  `json:"type"`
	Attributes *relationshipAttributes `json:"attributes"`
}

type relationshipAttributes struct {
	Name     string `json:"name"`
	FileName string `json:"fileName"`
}

const (
	baseURL      = "https://api.mangadex.org/manga"
	coverBaseURL = "https://uploads.mangadex.org/covers"
)

func main() {
	outputPath := flag.String("out", "data/manga_seed.json", "output JSON file")
	perDemo := flag.Int("per-demo", 25, "number of manga per demographic")
	flag.Parse()

	client := &http.Client{Timeout: 20 * time.Second}
	ctx := context.Background()

	demographics := []string{"shounen", "shoujo", "seinen", "josei"}
	allItems := make([]models.Manga, 0, (*perDemo)*len(demographics))
	seen := make(map[string]bool)

	for _, demo := range demographics {
		items, err := collectByDemographic(ctx, client, demo, *perDemo, seen)
		if err != nil {
			fmt.Printf("Failed to collect for %s: %v\n", demo, err)
			continue
		}
		allItems = append(allItems, items...)
		fmt.Printf("Collected %d for %s\n", len(items), demo)
	}

	targetTotal := (*perDemo) * len(demographics)
	if len(allItems) < targetTotal {
		need := targetTotal - len(allItems)
		fallback, err := collectFallback(ctx, client, need, seen)
		if err != nil {
			fmt.Printf("Fallback collection failed: %v\n", err)
		}
		allItems = append(allItems, fallback...)
	}

	payload, err := json.MarshalIndent(allItems, "", "  ")
	if err != nil {
		fmt.Printf("Failed to marshal JSON: %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile(*outputPath, payload, 0o644); err != nil {
		fmt.Printf("Failed to write file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Wrote %d items to %s\n", len(allItems), *outputPath)
}

func collectByDemographic(ctx context.Context, client *http.Client, demographic string, target int, seen map[string]bool) ([]models.Manga, error) {
	items := make([]models.Manga, 0, target)
	limit := 25
	offset := 0
	maxPages := 10

	for page := 0; page < maxPages && len(items) < target; page++ {
		resp, err := fetchPage(ctx, client, demographic, limit, offset)
		if err != nil {
			return items, err
		}
		if len(resp.Data) == 0 {
			break
		}

		for _, entry := range resp.Data {
			if seen[entry.ID] {
				continue
			}
			mapped := mapManga(entry)
			if mapped.ID == "" || mapped.Title == "" || mapped.Author == "" {
				continue
			}
			seen[mapped.ID] = true
			items = append(items, mapped)
			if len(items) >= target {
				break
			}
		}

		offset += limit
	}

	return items, nil
}

func collectFallback(ctx context.Context, client *http.Client, target int, seen map[string]bool) ([]models.Manga, error) {
	items := make([]models.Manga, 0, target)
	limit := 25
	offset := 0
	maxPages := 20

	for page := 0; page < maxPages && len(items) < target; page++ {
		resp, err := fetchPage(ctx, client, "", limit, offset)
		if err != nil {
			return items, err
		}
		if len(resp.Data) == 0 {
			break
		}

		for _, entry := range resp.Data {
			if seen[entry.ID] {
				continue
			}
			mapped := mapManga(entry)
			if mapped.ID == "" || mapped.Title == "" || mapped.Author == "" {
				continue
			}
			seen[mapped.ID] = true
			items = append(items, mapped)
			if len(items) >= target {
				break
			}
		}

		offset += limit
	}

	return items, nil
}

func fetchPage(ctx context.Context, client *http.Client, demographic string, limit, offset int) (*apiResponse, error) {
	endpoint, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}
	query := endpoint.Query()
	query.Set("limit", strconv.Itoa(limit))
	query.Set("offset", strconv.Itoa(offset))
	query.Add("includes[]", "author")
	query.Add("includes[]", "cover_art")
	query.Add("availableTranslatedLanguage[]", "en")
	query.Add("contentRating[]", "safe")
	query.Set("order[followedCount]", "desc")
	if demographic != "" {
		query.Add("publicationDemographic[]", demographic)
	}
	endpoint.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "MangaHubSeed/1.0 (educational)")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	var decoded apiResponse
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return nil, err
	}

	time.Sleep(350 * time.Millisecond)
	return &decoded, nil
}

func mapManga(item mangaItem) models.Manga {
	title := firstLocalized(item.Attributes.Title)
	description := firstLocalized(item.Attributes.Description)
	status := strings.TrimSpace(item.Attributes.Status)
	genres := extractGenres(item.Attributes.Tags)
	author := extractAuthor(item.Relationships)
	coverFile := extractCoverFile(item.Relationships)

	lastChapter := strings.TrimSpace(item.Attributes.LastChapter)
	totalChapters := 0
	if lastChapter != "" {
		if parsed, err := strconv.Atoi(lastChapter); err == nil {
			totalChapters = parsed
		}
	}

	mapped := models.Manga{
		ID:            item.ID,
		Title:         sanitizeASCII(title, "Untitled"),
		Author:        sanitizeASCII(author, "Unknown"),
		Genres:        sanitizeSlice(genres, "Unknown"),
		Status:        sanitizeASCII(status, "ongoing"),
		TotalChapters: totalChapters,
		Description:   sanitizeASCII(description, ""),
		CoverURL:      sanitizeASCII(buildCoverURL(item.ID, coverFile), ""),
	}

	if mapped.Title == "" {
		mapped.Title = item.ID
	}
	return mapped
}

func extractAuthor(relationships []mangaRelationship) string {
	for _, rel := range relationships {
		if rel.Type == "author" && rel.Attributes != nil {
			return rel.Attributes.Name
		}
	}
	return ""
}

func extractCoverFile(relationships []mangaRelationship) string {
	for _, rel := range relationships {
		if rel.Type == "cover_art" && rel.Attributes != nil {
			return rel.Attributes.FileName
		}
	}
	return ""
}

func extractGenres(tags []mangaTag) []string {
	genres := make([]string, 0)
	for _, tag := range tags {
		if tag.Attributes.Group != "genre" {
			continue
		}
		name := firstLocalized(tag.Attributes.Name)
		if name != "" {
			genres = append(genres, name)
		}
	}
	return genres
}

func firstLocalized(values map[string]string) string {
	if values == nil {
		return ""
	}
	if english := values["en"]; english != "" {
		return english
	}
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func buildCoverURL(mangaID, fileName string) string {
	if mangaID == "" || fileName == "" {
		return ""
	}
	return fmt.Sprintf("%s/%s/%s", coverBaseURL, mangaID, fileName)
}

func sanitizeSlice(values []string, fallback string) []string {
	clean := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := sanitizeASCII(value, "")
		if trimmed != "" {
			clean = append(clean, trimmed)
		}
	}
	if len(clean) == 0 && fallback != "" {
		return []string{fallback}
	}
	return clean
}

func sanitizeASCII(value, fallback string) string {
	if value == "" {
		return fallback
	}
	var b strings.Builder
	b.Grow(len(value))

	lastWasSpace := false
	for _, r := range value {
		switch {
		case r >= 32 && r <= 126:
			if r == ' ' {
				if lastWasSpace {
					continue
				}
				lastWasSpace = true
				b.WriteRune(r)
				continue
			}
			lastWasSpace = false
			b.WriteRune(r)
		case r == '\n' || r == '\r' || r == '\t':
			if lastWasSpace {
				continue
			}
			lastWasSpace = true
			b.WriteByte(' ')
		default:
			continue
		}
	}

	result := strings.TrimSpace(b.String())
	if result == "" {
		return fallback
	}
	return result
}
