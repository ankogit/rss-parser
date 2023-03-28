package app

import (
	"context"
	"github.com/jomei/notionapi"
	"github.com/k3a/html2text"
	"log"
	"strings"
)

func createNotionPage(client *notionapi.Client, item Item) {

	var techOptionsFilters []notionapi.Option
	for filter, _ := range item.Filters {
		techOptionsFilters = append(techOptionsFilters, notionapi.Option{
			Name: filter,
		})
	}
	var techOptions []notionapi.Option
	if len(item.Skills) == 0 {
		techOptions = techOptionsFilters
	}
	for _, filter := range item.Skills {
		techOptions = append(techOptions, notionapi.Option{
			Name: filter,
		})
	}

	country := "Russia"
	if len(item.Country) > 0 {
		country = item.Country
	}

	pageRequest := &notionapi.PageCreateRequest{Parent: notionapi.Parent{
		Type:       notionapi.ParentTypeDatabaseID,
		DatabaseID: dbId,
	},
		Properties: notionapi.Properties{
			"Название": notionapi.TitleProperty{
				Title: []notionapi.RichText{
					{Text: &notionapi.Text{Content: item.Title}},
				},
			},
		}}

	if len(item.Source) > 0 {
		pageRequest.Properties["Ресурс"] = notionapi.SelectProperty{
			Select: notionapi.Option{
				Name: item.Source,
			},
		}
	}

	pageRequest.Properties["Статус"] = notionapi.SelectProperty{
		Select: notionapi.Option{
			Name: newStatus,
		},
	}

	if len(techOptions) > 0 {
		pageRequest.Properties["Технологии"] = notionapi.MultiSelectProperty{
			MultiSelect: techOptions,
		}
	}
	if len(techOptionsFilters) > 0 {
		pageRequest.Properties["Фильтры"] = notionapi.MultiSelectProperty{
			MultiSelect: techOptionsFilters,
		}
	}
	if len(item.Link) > 0 {
		pageRequest.Properties["URL"] = notionapi.URLProperty{
			URL: item.Link,
		}
	}
	if len(item.Type) > 0 {
		pageRequest.Properties["Тип"] = notionapi.RichTextProperty{
			RichText: []notionapi.RichText{
				{
					Text: &notionapi.Text{
						Content: item.Type,
					},
					//PlainText: item.Type,
				},
			},
		}
	}
	if item.Budget != 0 {
		pageRequest.Properties["Бюджет"] = notionapi.NumberProperty{
			Number: float64(item.Budget),
		}
	}
	if len(country) > 0 {
		pageRequest.Properties["Страна"] = notionapi.SelectProperty{
			Select: notionapi.Option{
				Name: strings.Title(country),
			},
		}
	}
	if len(item.Hourly) > 0 {
		pageRequest.Properties["Hourly Range"] = notionapi.RichTextProperty{
			RichText: []notionapi.RichText{
				{
					Text: &notionapi.Text{
						Content: item.Hourly,
					},
				},
			},
		}
	}
	if item.HourlyFrom > 0 {
		pageRequest.Properties["Ставка от"] = notionapi.NumberProperty{
			Number: item.HourlyFrom,
		}
	}
	if item.HourlyTo > 0 {
		pageRequest.Properties["Ставка до"] = notionapi.NumberProperty{
			Number: item.HourlyTo,
		}
	}

	if len(item.Description) > 0 {
		pageRequest.Children = []notionapi.Block{
			notionapi.Heading1Block{
				BasicBlock: notionapi.BasicBlock{
					Object: notionapi.ObjectTypeBlock,
					Type:   notionapi.BlockTypeHeading1,
				},
				Heading1: notionapi.Heading{
					RichText: []notionapi.RichText{
						{
							Type: notionapi.ObjectTypeText,
							Text: &notionapi.Text{Content: "Background info"},
						},
					},
				},
			},
			notionapi.ParagraphBlock{
				BasicBlock: notionapi.BasicBlock{
					Object: notionapi.ObjectTypeBlock,
					Type:   notionapi.BlockTypeParagraph,
				},
				Paragraph: notionapi.Paragraph{
					RichText: []notionapi.RichText{
						{
							Text: &notionapi.Text{
								Content: html2text.HTML2Text(item.Description),
							},
						},
					},
					Children: nil,
				},
			},
		}
	}

	_, err := client.Page.Create(context.Background(), pageRequest)
	//notionapi.PageCreateRequest{
	//	Parent: notionapi.Parent{
	//		Type:       notionapi.ParentTypeDatabaseID,
	//		DatabaseID: dbId,
	//	},
	//	Properties: notionapi.Properties{
	//		"Название": notionapi.TitleProperty{
	//			Title: []notionapi.RichText{
	//				{Text: &notionapi.Text{Content: item.Title}},
	//			},
	//		},
	//		"Ресурс": notionapi.SelectProperty{
	//			Select: notionapi.Option{
	//				Name: item.Source,
	//			},
	//		},
	//		"Статус": notionapi.SelectProperty{
	//			Select: notionapi.Option{
	//				Name: newStatus,
	//			},
	//		},
	//		"Технологии": notionapi.MultiSelectProperty{
	//			MultiSelect: techOptions,
	//		},
	//		"Фильтры": notionapi.MultiSelectProperty{
	//			MultiSelect: techOptionsFilters,
	//		},
	//		"URL": notionapi.URLProperty{
	//			URL: item.Link,
	//		},
	//		"Тип": notionapi.RichTextProperty{
	//			RichText: []notionapi.RichText{
	//				{
	//					Text: &notionapi.Text{
	//						Content: item.Type,
	//					},
	//					//PlainText: item.Type,
	//				},
	//			},
	//		},
	//		"Бюджет": notionapi.NumberProperty{
	//			Number: float64(item.Budget),
	//		},
	//		"Страна": notionapi.SelectProperty{
	//			Select: notionapi.Option{
	//				Name: strings.Title(country),
	//			},
	//		},
	//
	//		"Hourly Range": notionapi.RichTextProperty{
	//			RichText: []notionapi.RichText{
	//				{
	//					Text: &notionapi.Text{
	//						Content: item.Hourly,
	//					},
	//				},
	//			},
	//		},
	//	},
	//	Children: []notionapi.Block{
	//		notionapi.Heading1Block{
	//			BasicBlock: notionapi.BasicBlock{
	//				Object: notionapi.ObjectTypeBlock,
	//				Type:   notionapi.BlockTypeHeading1,
	//			},
	//			Heading1: notionapi.Heading{
	//				RichText: []notionapi.RichText{
	//					{
	//						Type: notionapi.ObjectTypeText,
	//						Text: &notionapi.Text{Content: "Background info"},
	//					},
	//				},
	//			},
	//		},
	//		notionapi.ParagraphBlock{
	//			BasicBlock: notionapi.BasicBlock{
	//				Object: notionapi.ObjectTypeBlock,
	//				Type:   notionapi.BlockTypeParagraph,
	//			},
	//			Paragraph: notionapi.Paragraph{
	//				RichText: []notionapi.RichText{
	//					{
	//						Text: &notionapi.Text{
	//							Content: html2text.HTML2Text(item.Description),
	//						},
	//					},
	//				},
	//				Children: nil,
	//			},
	//		},
	//	},
	//})

	if err != nil {
		log.Println(err)
	}
	log.Println("Page Created")
}
