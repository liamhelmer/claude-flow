package github

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net/http"
	"time"

	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/go-github/v57/github"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	swarmv1alpha1 "github.com/claude-flow/swarm-operator/api/v1alpha1"
)

// TokenGenerator generates GitHub App installation tokens with repository restrictions
type TokenGenerator struct {
	client.Client
}

// NewTokenGenerator creates a new GitHub token generator
func NewTokenGenerator(client client.Client) *TokenGenerator {
	return &TokenGenerator{
		Client: client,
	}
}

// GenerateToken generates a GitHub App installation token for the given repositories
func (g *TokenGenerator) GenerateToken(ctx context.Context, appConfig *swarmv1alpha1.GitHubAppConfig, repositories []string, namespace string) (string, error) {
	log := log.FromContext(ctx)

	// Get the private key from the secret
	privateKey, err := g.getPrivateKey(ctx, appConfig.PrivateKeyRef, namespace)
	if err != nil {
		return "", fmt.Errorf("failed to get private key: %w", err)
	}

	// Create JWT for GitHub App authentication
	jwt, err := g.createAppJWT(appConfig.AppID, privateKey)
	if err != nil {
		return "", fmt.Errorf("failed to create JWT: %w", err)
	}

	// Create GitHub client with JWT
	tr := http.DefaultTransport
	client := github.NewClient(&http.Client{Transport: tr})
	client = client.WithAuthToken(jwt)

	// Get or find installation ID
	installationID := appConfig.InstallationID
	if installationID == 0 {
		log.Info("Finding GitHub App installation ID")
		installations, _, err := client.Apps.ListInstallations(ctx, &github.ListOptions{})
		if err != nil {
			return "", fmt.Errorf("failed to list installations: %w", err)
		}
		if len(installations) == 0 {
			return "", fmt.Errorf("no installations found for GitHub App")
		}
		// Use the first installation
		installationID = installations[0].GetID()
		log.Info("Found installation ID", "installationID", installationID)
	}

	// Create installation token with repository restrictions
	tokenOpts := &github.InstallationTokenOptions{}
	if len(repositories) > 0 {
		tokenOpts.Repositories = repositories
		// Set minimal required permissions
		tokenOpts.Permissions = &github.InstallationPermissions{
			Contents:     github.String("write"),
			PullRequests: github.String("write"),
			Issues:       github.String("write"),
			Actions:      github.String("read"),
			Metadata:     github.String("read"),
		}
	}

	token, _, err := client.Apps.CreateInstallationToken(ctx, installationID, tokenOpts)
	if err != nil {
		return "", fmt.Errorf("failed to create installation token: %w", err)
	}

	log.Info("Generated GitHub token", 
		"repositories", repositories,
		"expiresAt", token.GetExpiresAt())

	return token.GetToken(), nil
}

// getPrivateKey retrieves the private key from a Kubernetes secret
func (g *TokenGenerator) getPrivateKey(ctx context.Context, ref swarmv1alpha1.SecretKeyRef, defaultNamespace string) (*rsa.PrivateKey, error) {
	namespace := ref.Namespace
	if namespace == "" {
		namespace = defaultNamespace
	}

	secret := &corev1.Secret{}
	err := g.Get(ctx, types.NamespacedName{
		Name:      ref.Name,
		Namespace: namespace,
	}, secret)
	if err != nil {
		return nil, fmt.Errorf("failed to get secret: %w", err)
	}

	keyData, ok := secret.Data[ref.Key]
	if !ok {
		return nil, fmt.Errorf("key %s not found in secret", ref.Key)
	}

	// Parse PEM encoded private key
	block, _ := pem.Decode(keyData)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}

	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		// Try PKCS8 format
		keyInterface, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse private key: %w", err)
		}
		var ok bool
		key, ok = keyInterface.(*rsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("key is not RSA private key")
		}
	}

	return key, nil
}

// createAppJWT creates a JWT for GitHub App authentication
func (g *TokenGenerator) createAppJWT(appID int64, privateKey *rsa.PrivateKey) (string, error) {
	now := time.Now()
	claims := jwt.RegisteredClaims{
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(10 * time.Minute)),
		Issuer:    fmt.Sprintf("%d", appID),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	return token.SignedString(privateKey)
}

// CreateTokenSecret creates a Kubernetes secret containing the GitHub token
func (g *TokenGenerator) CreateTokenSecret(ctx context.Context, name, namespace, token string, repositories []string, expiresAt time.Time) error {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "swarm-operator",
				"swarm.claudeflow.io/type":     "github-token",
			},
			Annotations: map[string]string{
				"swarm.claudeflow.io/expires-at":    expiresAt.Format(time.RFC3339),
				"swarm.claudeflow.io/repositories":  strings.Join(repositories, ","),
			},
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"token": []byte(token),
		},
	}

	return g.Create(ctx, secret)
}

// UpdateTokenSecret updates an existing token secret
func (g *TokenGenerator) UpdateTokenSecret(ctx context.Context, name, namespace, token string, repositories []string, expiresAt time.Time) error {
	secret := &corev1.Secret{}
	err := g.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, secret)
	if err != nil {
		return err
	}

	secret.Data["token"] = []byte(token)
	secret.Annotations["swarm.claudeflow.io/expires-at"] = expiresAt.Format(time.RFC3339)
	secret.Annotations["swarm.claudeflow.io/repositories"] = strings.Join(repositories, ",")
	secret.Annotations["swarm.claudeflow.io/rotated-at"] = time.Now().Format(time.RFC3339)

	return g.Update(ctx, secret)
}

// IsTokenExpired checks if a token secret is expired
func (g *TokenGenerator) IsTokenExpired(ctx context.Context, name, namespace string) (bool, error) {
	secret := &corev1.Secret{}
	err := g.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, secret)
	if err != nil {
		return true, err
	}

	expiresAtStr, ok := secret.Annotations["swarm.claudeflow.io/expires-at"]
	if !ok {
		return true, nil
	}

	expiresAt, err := time.Parse(time.RFC3339, expiresAtStr)
	if err != nil {
		return true, err
	}

	// Consider token expired if it expires in less than 5 minutes
	return time.Now().Add(5 * time.Minute).After(expiresAt), nil
}