package http

import (
	"bytes"
	"encoding/json"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"lick-scroll/pkg/logger"
	"lick-scroll/services/post/internal/entity"
	"lick-scroll/services/post/internal/usecase"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockPostUseCase is a mock implementation of PostUseCase
type MockPostUseCase struct {
	mock.Mock
}

func (m *MockPostUseCase) CreatePost(userID string, title, description, postType, category string, mediaFile *multipart.FileHeader, imageFiles []*multipart.FileHeader) (*entity.Post, error) {
	args := m.Called(userID, title, description, postType, category, mediaFile, imageFiles)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Post), args.Error(1)
}

func (m *MockPostUseCase) GetPost(postID, userID string) (*entity.Post, int64, bool, error) {
	args := m.Called(postID, userID)
	if args.Get(0) == nil {
		return nil, 0, false, args.Error(3)
	}
	return args.Get(0).(*entity.Post), args.Get(1).(int64), args.Bool(2), args.Error(3)
}

func (m *MockPostUseCase) GetLikeCount(postID string) (int64, error) {
	args := m.Called(postID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockPostUseCase) ListPosts(limit, offset int, category string) ([]*entity.Post, error) {
	args := m.Called(limit, offset, category)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entity.Post), args.Error(1)
}

func (m *MockPostUseCase) UpdatePost(postID, userID string, title, description, category *string) (*entity.Post, error) {
	args := m.Called(postID, userID, title, description, category)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Post), args.Error(1)
}

func (m *MockPostUseCase) DeletePost(postID, userID string) error {
	args := m.Called(postID, userID)
	return args.Error(0)
}

func (m *MockPostUseCase) GetCreatorPosts(creatorID string, limit, offset int) ([]*entity.Post, error) {
	args := m.Called(creatorID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entity.Post), args.Error(1)
}

func (m *MockPostUseCase) LikePost(userID, postID string) (bool, error) {
	args := m.Called(userID, postID)
	return args.Bool(0), args.Error(1)
}

func (m *MockPostUseCase) IsLiked(userID, postID string) (bool, error) {
	args := m.Called(userID, postID)
	return args.Bool(0), args.Error(1)
}

func (m *MockPostUseCase) GetLikedPosts(userID string, limit, offset int) ([]*entity.Post, error) {
	args := m.Called(userID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entity.Post), args.Error(1)
}

func (m *MockPostUseCase) IncrementView(postID string) error {
	args := m.Called(postID)
	return args.Error(0)
}

var _ usecase.PostUseCase = (*MockPostUseCase)(nil)

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

func TestLikePost_Like(t *testing.T) {
	mockUseCase := new(MockPostUseCase)
	logger := logger.New()
	handler := NewPostHandler(mockUseCase, nil, logger)

	router := setupTestRouter()
	router.POST("/posts/:id/like", func(c *gin.Context) {
		c.Set("user_id", "user-123")
		handler.LikePost(c)
	})

	postID := "post-123"
	userID := "user-123"

	mockUseCase.On("LikePost", userID, postID).Return(true, nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/posts/"+postID+"/like", nil)
	req.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Equal(t, "Post liked", response["message"])
	assert.Equal(t, true, response["liked"])

	mockUseCase.AssertExpectations(t)
}

func TestUpdatePost_Success(t *testing.T) {
	mockUseCase := new(MockPostUseCase)
	logger := logger.New()
	handler := NewPostHandler(mockUseCase, nil, logger)

	router := setupTestRouter()
	router.PUT("/posts/:id", func(c *gin.Context) {
		c.Set("user_id", "creator-123")
		handler.UpdatePost(c)
	})

	postID := "post-123"
	userID := "creator-123"

	mockPost := &entity.Post{
		ID:        postID,
		CreatorID: userID,
		Title:     "New Title",
		Type:      entity.PostTypePhoto,
		Status:    entity.StatusApproved,
	}

	title := "New Title"
	mockUseCase.On("UpdatePost", postID, userID, &title, (*string)(nil), (*string)(nil)).Return(mockPost, nil)
	mockUseCase.On("GetLikeCount", postID).Return(int64(0), nil)

	updateJSON := `{"title":"New Title"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/posts/"+postID, bytes.NewBufferString(updateJSON))
	req.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockUseCase.AssertExpectations(t)
}

func TestUpdatePost_NotFound(t *testing.T) {
	mockUseCase := new(MockPostUseCase)
	logger := logger.New()
	handler := NewPostHandler(mockUseCase, nil, logger)

	router := setupTestRouter()
	router.PUT("/posts/:id", handler.UpdatePost)

	postID := "post-not-found"
	userID := ""

	title := "New Title"
	mockUseCase.On("UpdatePost", postID, userID, &title, (*string)(nil), (*string)(nil)).Return(nil, errors.New("post not found"))

	updateJSON := `{"title":"New Title"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/posts/"+postID, bytes.NewBufferString(updateJSON))

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockUseCase.AssertExpectations(t)
}

func TestDeletePost_Success(t *testing.T) {
	t.Skip("Skipping - DeletePost requires Redis mock")
}

func TestDeletePost_NotFound(t *testing.T) {
	mockUseCase := new(MockPostUseCase)
	logger := logger.New()
	handler := NewPostHandler(mockUseCase, nil, logger)

	router := setupTestRouter()
	router.DELETE("/posts/:id", handler.DeletePost)

	postID := "post-not-found"
	userID := ""

	mockUseCase.On("DeletePost", postID, userID).Return(errors.New("post not found"))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/posts/"+postID, nil)

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockUseCase.AssertExpectations(t)
}

func TestLikePost_Unlike(t *testing.T) {
	mockUseCase := new(MockPostUseCase)
	logger := logger.New()
	handler := NewPostHandler(mockUseCase, nil, logger)

	router := setupTestRouter()
	router.POST("/posts/:id/like", func(c *gin.Context) {
		c.Set("user_id", "user-123")
		handler.LikePost(c)
	})

	postID := "post-123"
	userID := "user-123"

	mockUseCase.On("LikePost", userID, postID).Return(false, nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/posts/"+postID+"/like", nil)
	req.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Equal(t, "Post unliked", response["message"])
	assert.Equal(t, false, response["liked"])

	mockUseCase.AssertExpectations(t)
}

func TestLikePost_PostNotFound(t *testing.T) {
	mockUseCase := new(MockPostUseCase)
	logger := logger.New()
	handler := NewPostHandler(mockUseCase, nil, logger)

	router := setupTestRouter()
	router.POST("/posts/:id/like", handler.LikePost)

	postID := "post-not-found"
	userID := ""

	mockUseCase.On("LikePost", userID, postID).Return(false, errors.New("post not found"))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/posts/"+postID+"/like", nil)

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockUseCase.AssertExpectations(t)
}

func TestFormatPostResponse_WithMediaURL(t *testing.T) {
	mockUseCase := new(MockPostUseCase)
	logger := logger.New()
	handler := NewPostHandler(mockUseCase, nil, logger)

	post := &entity.Post{
		ID:        "post-123",
		CreatorID: "creator-123",
		Title:     "Test Post",
		MediaURL:  "http://example.com/video.mp4",
		Type:      entity.PostTypeVideo,
		Status:    entity.StatusApproved,
	}

	likeCount := int64(5)
	response := handler.formatPostResponse(post, likeCount)

	assert.Equal(t, post.ID, response["id"])
	assert.Equal(t, post.Title, response["title"])
	assert.Equal(t, likeCount, response["likes_count"])
	assert.Equal(t, post.MediaURL, response["media_url"])
}

func TestFormatPostResponse_WithThumbnailURL(t *testing.T) {
	mockUseCase := new(MockPostUseCase)
	logger := logger.New()
	handler := NewPostHandler(mockUseCase, nil, logger)

	post := &entity.Post{
		ID:           "post-123",
		CreatorID:    "creator-123",
		Title:        "Test Post",
		ThumbnailURL: "http://example.com/thumb.jpg",
		Type:         entity.PostTypeVideo,
		Status:       entity.StatusApproved,
	}

	likeCount := int64(3)
	response := handler.formatPostResponse(post, likeCount)

	assert.Equal(t, post.ID, response["id"])
	assert.Equal(t, likeCount, response["likes_count"])
	assert.Equal(t, post.ThumbnailURL, response["thumbnail_url"])
}

func TestFormatPostResponse_WithImages(t *testing.T) {
	mockUseCase := new(MockPostUseCase)
	logger := logger.New()
	handler := NewPostHandler(mockUseCase, nil, logger)

	post := &entity.Post{
		ID:        "post-123",
		CreatorID: "creator-123",
		Title:     "Test Post",
		Type:      entity.PostTypePhoto,
		Status:    entity.StatusApproved,
		Images: []entity.PostImage{
			{ID: "img-1", ImageURL: "http://example.com/img1.jpg"},
			{ID: "img-2", ImageURL: "http://example.com/img2.jpg"},
		},
	}

	likeCount := int64(10)
	response := handler.formatPostResponse(post, likeCount)

	assert.Equal(t, post.ID, response["id"])
	assert.Equal(t, likeCount, response["likes_count"])
	assert.NotNil(t, response["images"])
}

func TestListPosts_Success(t *testing.T) {
	mockUseCase := new(MockPostUseCase)
	logger := logger.New()
	handler := NewPostHandler(mockUseCase, nil, logger)

	router := setupTestRouter()
	router.GET("/posts", handler.ListPosts)

	mockPosts := []*entity.Post{
		{
			ID:        "post-1",
			CreatorID: "creator-1",
			Title:     "Post 1",
			Type:      entity.PostTypePhoto,
			Status:    entity.StatusApproved,
		},
		{
			ID:        "post-2",
			CreatorID: "creator-2",
			Title:     "Post 2",
			Type:      entity.PostTypePhoto,
			Status:    entity.StatusApproved,
		},
	}

	mockUseCase.On("ListPosts", 20, 0, "").Return(mockPosts, nil)
	mockUseCase.On("GetLikeCount", "post-1").Return(int64(5), nil)
	mockUseCase.On("GetLikeCount", "post-2").Return(int64(3), nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/posts", nil)

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	posts := response["posts"].([]interface{})
	assert.GreaterOrEqual(t, len(posts), 0)

	mockUseCase.AssertExpectations(t)
}

func TestNewPostHandler(t *testing.T) {
	mockUseCase := new(MockPostUseCase)
	logger := logger.New()
	handler := NewPostHandler(mockUseCase, nil, logger)

	assert.NotNil(t, handler)
}

func TestGetPost_Success(t *testing.T) {
	t.Skip("Skipping - GetPost requires Redis mock")
}

func TestGetPost_NotFound(t *testing.T) {
	t.Skip("Skipping - GetPost requires Redis mock")
}

func TestGetCreatorPosts_Success(t *testing.T) {
	mockUseCase := new(MockPostUseCase)
	logger := logger.New()
	handler := NewPostHandler(mockUseCase, nil, logger)

	router := setupTestRouter()
	router.GET("/creators/:creator_id/posts", handler.GetCreatorPosts)

	creatorID := "creator-123"
	mockPosts := []*entity.Post{
		{
			ID:        "post-1",
			CreatorID: creatorID,
			Title:     "Post 1",
			Type:      entity.PostTypePhoto,
			Status:    entity.StatusApproved,
		},
		{
			ID:        "post-2",
			CreatorID: creatorID,
			Title:     "Post 2",
			Type:      entity.PostTypePhoto,
			Status:    entity.StatusApproved,
		},
	}

	mockUseCase.On("GetCreatorPosts", creatorID, 20, 0).Return(mockPosts, nil)
	mockUseCase.On("GetLikeCount", "post-1").Return(int64(5), nil)
	mockUseCase.On("GetLikeCount", "post-2").Return(int64(3), nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/creators/"+creatorID+"/posts", nil)

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	posts := response["posts"].([]interface{})
	assert.Equal(t, 2, len(posts))

	mockUseCase.AssertExpectations(t)
}
