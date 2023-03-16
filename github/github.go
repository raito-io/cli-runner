package github

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"runtime"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/google/go-github/v50/github"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/sirupsen/logrus"
)

const RAITO_CLI_REPOSITORY_OWNER = "raito-io"
const RAITO_CLI_REPOSITORY_NAME = "cli"

type GithubRepo struct {
	httpClient    *retryablehttp.Client
	client        *github.Client
	releaseSuffix string
}

func NewGithubRepo() *GithubRepo {
	httpClient := retryablehttp.NewClient()
	httpClient.Logger = logrus.StandardLogger()

	return &GithubRepo{
		httpClient:    httpClient,
		client:        github.NewClient(httpClient.StandardClient()),
		releaseSuffix: runtime.GOOS + "_" + runtime.GOARCH + ".tar.gz",
	}
}

func (g *GithubRepo) GetLatestReleasedVersion(ctx context.Context) (*semver.Version, error) {
	release, _, err := g.client.Repositories.GetLatestRelease(ctx, RAITO_CLI_REPOSITORY_OWNER, RAITO_CLI_REPOSITORY_NAME)
	if err != nil {
		return nil, fmt.Errorf("get latest version: %w", err)
	}

	return semver.NewVersion(release.GetTagName())
}

func (g *GithubRepo) InstallLatestRelease(ctx context.Context, dir string) (*semver.Version, string, error) {
	release, _, err := g.client.Repositories.GetLatestRelease(ctx, RAITO_CLI_REPOSITORY_OWNER, RAITO_CLI_REPOSITORY_NAME)
	if err != nil {
		return nil, "", fmt.Errorf("get latest version: %w", err)
	}

	version, err := semver.NewVersion(release.GetTagName())
	if err != nil {
		return nil, "", err
	}

	var correctAsset *github.ReleaseAsset

	for _, asset := range release.Assets {
		if strings.HasSuffix(asset.GetName(), g.releaseSuffix) {
			correctAsset = asset
			break
		}
	}

	if correctAsset == nil {
		return nil, "", errors.New("no compatible asset found")
	}

	location, err := g.downloadGitHubAsset(correctAsset.GetBrowserDownloadURL(), version)
	if err != nil {
		return nil, "", err
	}

	location, err = extractFromDownloadFile(location, dir, version)
	if err != nil {
		return nil, "", err
	}

	return version, location, nil
}

func (g *GithubRepo) downloadGitHubAsset(url string, version *semver.Version) (string, error) {
	resp, err := g.httpClient.Get(url)
	if err != nil {
		return "", fmt.Errorf("error while fetching release asset from %q: %s", url, err.Error())
	}

	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("error while fetching release asset from %q", url)
	}

	defer resp.Body.Close()

	// Create the file
	out, err := os.CreateTemp("", "raito-cli-"+version.String()+".tar.gz")
	if err != nil {
		return "", fmt.Errorf("error while creating temporary file for asset download: %s", err.Error())
	}
	defer out.Close()

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return out.Name(), fmt.Errorf("error while storing release asset from %q to temporary file: %s", url, err.Error())
	}

	return out.Name(), nil
}

func extractFromDownloadFile(downloadedFile, targetPath string, version *semver.Version) (string, error) {
	tarGzFile, err := os.Open(downloadedFile)
	if err != nil {
		return "", fmt.Errorf("error while reading tar.gz archive %s: %s", downloadedFile, err.Error())
	} else {
		defer tarGzFile.Close()
		extractedFile := targetPath + "cli-" + version.String()
		extractedFile, err := extractTarGz(tarGzFile, extractedFile)

		if err != nil {
			return "", fmt.Errorf("error while reading tar.gz archive %s: %s", downloadedFile, err.Error())
		} else {
			if err := os.Chmod(extractedFile, 0750); err != nil {
				return "", fmt.Errorf("error while setting the right permissions for plugin file %q: %s", extractedFile, err.Error())
			}
			return extractedFile, nil
		}
	}
}

func extractTarGz(gzipStream io.Reader, extractedPath string) (string, error) {
	parentFolder := extractedPath[0 : strings.LastIndex(extractedPath, "/")+1]

	err := os.MkdirAll(parentFolder, fs.ModePerm)
	if err != nil {
		return "", fmt.Errorf("error while creating plugin parent folder %q: %s", parentFolder, err.Error())
	}

	uncompressedStream, err := gzip.NewReader(gzipStream)
	if err != nil {
		return "", fmt.Errorf("error while reading gzip stream: %s", err.Error())
	}

	tarReader := tar.NewReader(uncompressedStream)

	for {
		header, err := tarReader.Next()

		if err == io.EOF {
			break
		}

		if err != nil {
			return "", fmt.Errorf("error while extracting file from tar.gz archive: %s", err.Error())
		}

		switch header.Typeflag {
		case tar.TypeDir:
			return "", fmt.Errorf("found directories in the tar.gz archive")
		case tar.TypeReg:
			// goreleaser will also include the LICENSE AND README files, we ignore them.
			// we also ignore other files that are 1MB as they cannot be the binary we are looking for
			if header.Name == "LICENSE" || header.Name == "README" || header.Size < 1024*1024 {
				continue
			}
			outFile, err := os.Create(extractedPath)

			if err != nil {
				return "", fmt.Errorf("error while extracting file from tar.gz archive: %s", err.Error())
			}

			for {
				if _, err := io.CopyN(outFile, tarReader, 1024); err != nil {
					if err != nil {
						if err == io.EOF {
							break
						}

						return "", fmt.Errorf("error while extracting file from tar.gz archive: %s", err.Error())
					}
				}
			}

			outFile.Close()

			return extractedPath, nil
		default:
			return "", errors.New("unknown entry found in tar.gz archive")
		}
	}

	return "", errors.New("no files found to extract from tar.gz archive")
}
