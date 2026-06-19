package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type ArticleLD struct {
	Context     string    `json:"@context"`
	Type        string    `json:"@type"`
	Headline    string    `json:"headline"`
	Description string    `json:"description"`
	DatePublished string  `json:"datePublished"`
	Author      AuthorLD  `json:"author"`
	Publisher   PublisherLD `json:"publisher"`
}

type AuthorLD struct {
	Type string `json:"@type"`
	Name string `json:"name"`
}

type PublisherLD struct {
	Type string `json:"@type"`
	Name string `json:"name"`
	Logo LogoLD  `json:"logo"`
}

type LogoLD struct {
	Type string `json:"@type"`
	URL  string `json:"url"`
}

type BreadcrumbLD struct {
	Context        string          `json:"@context"`
	Type           string          `json:"@type"`
	ItemListElement []BreadcrumbItem `json:"itemListElement"`
}

type BreadcrumbItem struct {
	Type     string `json:"@type"`
	Position int    `json:"position"`
	Name     string `json:"name"`
	Item     string `json:"item"`
}

type ProductLD struct {
	Context        string  `json:"@context"`
	Type           string  `json:"@type"`
	Name           string  `json:"name"`
	Description    string  `json:"description"`
	Image          string  `json:"image"`
	Offers         OfferLD `json:"offers"`
	AggregateRating *RatingLD `json:"aggregateRating,omitempty"`
}

type OfferLD struct {
	Type        string  `json:"@type"`
	Price       float64 `json:"price"`
	PriceCurrency string `json:"priceCurrency"`
	URL         string  `json:"url"`
	Availability string `json:"availability"`
}

type RatingLD struct {
	Type           string  `json:"@type"`
	RatingValue    float64 `json:"ratingValue"`
	ReviewCount    int     `json:"reviewCount"`
	BestRating     float64 `json:"bestRating"`
	WorstRating    float64 `json:"worstRating"`
}

type FAQPageLD struct {
	Context     string    `json:"@context"`
	Type        string    `json:"@type"`
	MainEntity  []FAQItem `json:"mainEntity"`
}

type FAQItem struct {
	Type     string `json:"@type"`
	Name     string `json:"name"`
	AcceptedAnswer AnswerLD `json:"acceptedAnswer"`
}

type AnswerLD struct {
	Type string `json:"@type"`
	Text string `json:"text"`
}

func jsonLDScript(data interface{}) string {
	b, _ := json.MarshalIndent(data, "", "  ")
	return fmt.Sprintf(`<script type="application/ld+json">%s</script>`, string(b))
}

func articleJSONLD(title, desc string) string {
	return jsonLDScript(ArticleLD{
		Context:        "https://schema.org",
		Type:           "Article",
		Headline:       title,
		Description:    desc,
		DatePublished:  time.Now().Format("2006-01-02"),
		Author:         AuthorLD{Type: "Organization", Name: "Emerald Engine"},
		Publisher:      PublisherLD{Type: "Organization", Name: "Emerald Engine", Logo: LogoLD{Type: "ImageObject", URL: baseURL + "/logo.png"}},
	})
}

func productJSONLD(name, desc string, price float64, url string) string {
	return jsonLDScript(ProductLD{
		Context:     "https://schema.org",
		Type:        "Product",
		Name:        name,
		Description: desc,
		Image:       baseURL + "/images/product-default.png",
		Offers: OfferLD{
			Type:          "Offer",
			Price:         price,
			PriceCurrency: "USD",
			URL:           url,
			Availability: "https://schema.org/InStock",
		},
		AggregateRating: &RatingLD{
			Type:        "AggregateRating",
			RatingValue: 4.5,
			ReviewCount: 127,
			BestRating:  5,
			WorstRating: 1,
		},
	})
}

func breadcrumbJSONLD(items []BreadcrumbItem) string {
	return jsonLDScript(BreadcrumbLD{
		Context: "https://schema.org",
		Type:    "BreadcrumbList",
		ItemListElement: items,
	})
}

func faqJSONLD(questions []struct{ Q, A string }) string {
	var items []FAQItem
	for _, q := range questions {
		items = append(items, FAQItem{
			Type: "Question",
			Name: q.Q,
			AcceptedAnswer: AnswerLD{
				Type: "Answer",
				Text: q.A,
			},
		})
	}
	return jsonLDScript(FAQPageLD{
		Context:    "https://schema.org",
		Type:       "FAQPage",
		MainEntity: items,
	})
}

func openGraphTags(title, desc, pageURL, image string) string {
	if image == "" {
		image = baseURL + "/og-default.png"
	}
	return fmt.Sprintf(`<meta property="og:title" content="%s">
<meta property="og:description" content="%s">
<meta property="og:url" content="%s">
<meta property="og:type" content="website">
<meta property="og:image" content="%s">
<meta name="twitter:card" content="summary_large_image">
<meta name="twitter:title" content="%s">
<meta name="twitter:description" content="%s">
<meta name="twitter:image" content="%s">`,
		escapeHTML(title), escapeHTML(desc), pageURL, image,
		escapeHTML(title), escapeHTML(desc), image,
	)
}

func escapeHTML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "'", "&#39;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}

func commonHeadSEO(title, desc, pagePath string) string {
	pageURL := baseURL + pagePath
	return fmt.Sprintf(`<!DOCTYPE html><html lang="en"><head><meta charset="UTF-8"><meta name="viewport" content="width=device-width,initial-scale=1.0"><title>%s</title><meta name="description" content="%s"><meta name="robots" content="index,follow"><link rel="canonical" href="%s">
%s
%s
%s
<style>%s</style>%s</head><body><div class="container">`,
		title, desc, pageURL,
		gaTag(),
		openGraphTags(title, desc, pageURL, ""),
		articleJSONLD(title, desc),
		commonCSS, toolJS,
	)
}
