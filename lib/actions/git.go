package actions

import (
	"fmt"
	"github.com/alecthomas/colour"
	gogit "gopkg.in/src-d/go-git.v4"
	gogitConfig "gopkg.in/src-d/go-git.v4/config"
	"log"
	"os"
	"sopr/lib/git"
	"sopr/lib/prompts"
	"sort"
	"sopr/lib/config"
)

func GitInitialize() {
	repos, err := git.RepoList(true)

	if err != nil {
		log.Fatalf("Error: could not read repository list - %s", err)
	}

	for _, repo := range repos {
		colour.Printf("Cloning into ^2%s^R. \n", repo.Config.Name)
		fmt.Println("")

		if(repo.Config.Remotes == nil || len(repo.Config.Remotes) == 0) {
			colour.Printf("No remotes configured for ^2%s^R. \n", repo.Config.Name)
			break;
		}

		// determine which remote should be the main remote
		// first attempt to see if an "origin" is provided
		// if no "origin" is found, use the first remote
		var mainRemote *config.Remote
		for _, remote := range repo.Config.Remotes {
			if (remote.Name == "origin") {
				mainRemote = &remote
				break
			}
		}

		if (mainRemote == nil) {
			mainRemote = &repo.Config.Remotes[0]
		}

		ref, err := gogit.PlainClone(repo.FullPath, false, &gogit.CloneOptions{
			URL:      mainRemote.Url,
			RemoteName: mainRemote.Name,
			Progress: os.Stdout,
		})

		for _, remote := range repo.Config.Remotes {
			if (mainRemote.Name == remote.Name) {
				continue
			}

			remoteConfig := &gogitConfig.RemoteConfig{
				Name: remote.Name,
				URLs: []string{remote.Url},
			}

			if _, err := ref.CreateRemote(remoteConfig); err != nil {
				colour.Printf("Could not configure remote ^2%s^R. \n", remote.Url)
			}
		}

		fmt.Println("")
		if err != nil {
			fmt.Println(fmt.Sprintf("Warning: could not clone repo (%s) - %s", repo.Config.Name, err))
		} else {
			colour.Printf("^2%s^R cloned to ^2%s^R. \n", repo.Config.Name, repo.FullPath)
		}

		fmt.Println("")
	}
}

func GitCheckoutBranch(branchName string, allRepos bool, create bool) {
	var selectedRepos []git.Repo
	if branchName == "" {
		branchName = prompts.BranchNamePrompt()
	}

	repos, err := git.RepoList(false)
	if err != nil {
		fmt.Printf("Error getting repo list %s", err)
		os.Exit(1)
	}

	if allRepos {
		selectedRepos = repos
	} else {
		selectedRepos = prompts.RepoSelectPrompt(repos)
	}

	pristineRepos := getPristineRepos(selectedRepos, repos)

	for _, repo := range pristineRepos {
		colour.Printf("Checking out Branch ^4%s^R in: ^2%s^R. \n", branchName, repo.Config.Name)

		err := repo.Checkout(branchName, create)
		if err != nil {
			fmt.Println(fmt.Sprintf("Error: %s - %s", repo.Config.Name, err))
		}
	}
}

func GitListRepos() {
	var output []map[string]string

	repos, err := git.RepoList(false)
	if err != nil {
		fmt.Printf("Error: can't list repos %s", err)
	}

	for _, repo := range repos {
		branch, err := repo.Branch()
		if err != nil {
			fmt.Printf("Error: could not get branch for %s", repo.Config.Name)
			continue
		}

		output = append(output, map[string]string{
			"Name":   repo.Config.Name,
			"Branch": branch,
		})
	}

	sort.Slice(output, func(i, j int) bool { return output[i]["Name"] < output[j]["Name"] })

	for _, repo := range output {
		colour.Printf("^2%s^R (^4%s^R) \n", repo["Name"], repo["Branch"])
	}
}

func getPristineRepos(selectedRepos []git.Repo, repoList []git.Repo) []git.Repo {
	var pristineRepos []git.Repo

	existingNameMap := make(map[string]bool)

	for _, repo := range selectedRepos {
		existingNameMap[repo.Config.Name] = true
	}

	fmt.Println("Checking local working tree status")
	for _, repo := range repoList {
		if existingNameMap[repo.Config.Name] {
			if !repo.IsClean() {
				colour.Printf("Working tree for ^2%s^R is not clean, skipping. \n", repo.Config.Name)
				continue
			}

			pristineRepos = append(pristineRepos, repo)
		}
	}

	return pristineRepos
}

func GitUpdate(allRepos bool) {
	var selectedRepos []git.Repo

	repos, err := git.RepoList(false)
	if err != nil {
		fmt.Printf("Error getting repo list %s", err)
		os.Exit(1)
	}

	if allRepos {
		selectedRepos = repos
	} else {
		selectedRepos = prompts.RepoSelectPrompt(repos)
	}

	pristineRepos := getPristineRepos(selectedRepos, repos)

	for _, repo := range pristineRepos {
		err := repo.Pull()
		if err != nil && err.Error() == "already up-to-date" {
			colour.Printf("Skipping ^2%s^R because its already up to date. \n", repo.Config.Name)
			continue
		} else if err != nil {
			fmt.Println(fmt.Sprintf("Error: %s - %s", repo.Config.Name, err))
			continue
		}

		colour.Printf("Updating ^2%s^R. \n", repo.Config.Name)
		fmt.Println(fmt.Sprintf("Updating %s", repo.Config.Name))
	}
}
