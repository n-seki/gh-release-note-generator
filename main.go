package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/shurcooL/githubv4"

	"golang.org/x/oauth2"
)

type Label struct {
	Name string
}

type LabelConnection struct {
	Nodes []Label
}

type Repository struct {
	Name string
}

type Actor struct {
	Login string
}

type IssueTimelineItems struct {
	CrossReferencedEvent struct {
		ReferencedSubject struct {
			PullRequest struct {
				Title      string
				Author     Actor
				Number     int
				Merged     bool
				Repository Repository
			} `graphql:"... on PullRequest"`
		} `graphql:"source"`
	} `graphql:"... on CrossReferencedEvent"`
}

type IssueTimelineItemsConnection struct {
	Nodes []IssueTimelineItems
}

type ProjectV2Item struct {
	Content struct {
		Issue struct {
			Title         string
			Number        int
			Labels        LabelConnection              `graphql:"labels(first: 10)"`
			TimelineItems IssueTimelineItemsConnection `graphql:"timelineItems(first: 100, itemTypes: $timelineItemsTypes)"`
		} `graphql:"... on Issue"`
	}
}

type ProjectV2ItemConnection struct {
	Nodes []ProjectV2Item
}

type ProjectPullRequestsQuery struct {
	Organization struct {
		ProjectV2 struct {
			Items ProjectV2ItemConnection `graphql:"items(first: $maxItemCount)"`
		} `graphql:"projectV2(number: $projectId)"`
	} `graphql:"organization(login: $login)"`
}

type ReleaseNoteItem struct {
	issueTitle   string
	prShortLinks []string
	authors      []string
}

var cmd = &cobra.Command{
	Use:     "./gh-release-note-generator -o 'organization' -r 'repository' -p 'project' -l 'label name'",
	Version: "0.0.1",
	Run: func(cmd *cobra.Command, args []string) {
		token, err := cmd.Flags().GetString("token")
		if err != nil {
			log.Fatal(err)
		}
		if len(token) == 0 {
			token = os.Getenv("GITHUB_ACCESS_TOKEN")
		}
		if len(token) == 0 {
			log.Fatal(fmt.Errorf("missing GitHub Access Token"))
		}
		project, err := cmd.Flags().GetInt("project")
		if err != nil {
			log.Fatal(err)
		}
		orginization, err := cmd.Flags().GetString("organization")
		if err != nil {
			log.Fatal(err)
		}
		repository, err := cmd.Flags().GetString("repository")
		if err != nil {
			log.Fatal(err)
		}
		labels, err := cmd.Flags().GetStringArray("labels")
		if err != nil {
			log.Fatal(err)
		}
		maxItemCount, err := cmd.Flags().GetInt("max-item-count")
		if err != nil {
			log.Fatal(err)
		}
		excludeLabels, err := cmd.Flags().GetStringArray("exclude-labels")
		if err != nil {
			log.Fatal(err)
		}
		generateGitHubReleaseNote(token, project, orginization, repository, labels, excludeLabels, maxItemCount)
	},
}

func init() {
	cobra.OnInitialize()
	cmd.PersistentFlags().StringP("token", "t", "", "GitHub access token")
	cmd.PersistentFlags().StringP("organization", "o", "", "Organization")
	cmd.MarkPersistentFlagRequired("organization")
	cmd.PersistentFlags().IntP("project", "p", 1, "Target GitHub Project ID")
	cmd.MarkPersistentFlagRequired("project")
	cmd.PersistentFlags().StringP("repository", "r", "", "Target repository")
	cmd.MarkPersistentFlagRequired("repository")
	cmd.PersistentFlags().StringArrayP("labels", "l", []string{}, "Target issue labels")
	cmd.MarkPersistentFlagRequired("labels")
	cmd.PersistentFlags().IntP("max-item-count", "i", 100, "Max item count in target Project")
	cmd.PersistentFlags().StringArray("exclude-labels", []string{}, "Exclude issue labels")
}

func main() {
	cmd.Execute()
}

func generateGitHubReleaseNote(
	token string,
	projectId int,
	organization string,
	repository string,
	labels []string,
	excludeLabels []string,
	maxItemCount int,
) {
	src := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	httpClient := oauth2.NewClient(context.Background(), src)
	graphQLClient := githubv4.NewClient(httpClient)

	variables := map[string]interface{}{
		"login":              githubv4.String(organization),
		"projectId":          githubv4.Int(projectId),
		"maxItemCount":       githubv4.Int(maxItemCount),
		"timelineItemsTypes": []githubv4.IssueTimelineItemsItemType{githubv4.IssueTimelineItemsItemTypeCrossReferencedEvent},
	}

	var query ProjectPullRequestsQuery

	err := graphQLClient.Query(context.Background(), &query, variables)
	if err != nil {
		log.Fatal(err)
	}

	releaseNoteItems := make(map[string][]ReleaseNoteItem)

	for _, node := range query.Organization.ProjectV2.Items.Nodes {
		label := ""
		skip := false
		for _, l := range node.Content.Issue.Labels.Nodes {
			if slices.Contains(excludeLabels, l.Name) {
				skip = true
				break
			}
			if slices.Contains(labels, l.Name) {
				label = l.Name
				break
			}
		}
		if skip || len(label) == 0 {
			continue
		}
		prLinks := []string{}
		authors := []string{}
		for _, node := range node.Content.Issue.TimelineItems.Nodes {
			pr := node.CrossReferencedEvent.ReferencedSubject.PullRequest
			repo := pr.Repository
			if repo.Name == repository && pr.Merged {
				number := pr.Number
				shortLink := "#" + strconv.Itoa(number)
				prLinks = append(prLinks, shortLink)

				author := "@" + pr.Author.Login
				if !slices.Contains(authors, author) {
					authors = append(authors, author)
				}
			}
		}
		if len(prLinks) == 0 {
			continue
		}
		item := ReleaseNoteItem{
			issueTitle:   node.Content.Issue.Title,
			prShortLinks: prLinks,
			authors:      authors,
		}
		if _, ok := releaseNoteItems[label]; !ok {
			releaseNoteItems[label] = []ReleaseNoteItem{}
		}
		releaseNoteItems[label] = append(releaseNoteItems[label], item)
	}

	releaseNote := ""

	for _, label := range labels {
		items := releaseNoteItems[label]
		if len(items) == 0 {
			continue
		}
		releaseNote += "## " + label + "\n"
		for _, item := range items {
			authors := strings.Join(item.authors, ", ")
			links := strings.Join(item.prShortLinks, ", ")
			releaseNote += fmt.Sprintf("* %s by %s in %s\n", item.issueTitle, authors, links)
		}
		releaseNote += "\n"
	}

	fmt.Println(releaseNote)
}
