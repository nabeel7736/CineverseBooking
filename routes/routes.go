package routes

import (
	"cineverse/config"
	"cineverse/controllers"
	"cineverse/middlewares"

	"github.com/gin-gonic/gin"
)

func SetupRouter() *gin.Engine {
	r := gin.Default()

	// Load all HTML templates
	r.LoadHTMLGlob("templates/*")

	// Public API Routes

	api := r.Group("/api")
	{
		// Auth routes
		api.POST("/signup", controllers.SignupHandler)
		api.POST("/login", controllers.LoginHandler)
		api.POST("/forgot-password", controllers.ForgotPasswordHandler)
		api.POST("/reset-password", controllers.ResetPasswordHandler)

		// New refresh token endpoints
		api.POST("/refresh", controllers.RefreshTokenHandler)
		api.POST("/logout", controllers.LogoutHandler)

		// Public movie routes
		api.GET("/movies", controllers.GetMovies(config.DB))
		api.GET("/movies/:id", controllers.GetMovieDetails(config.DB))
		api.GET("/movies/:id/shows", controllers.GetShowsByMovie(config.DB))
	}

	// Protected User Routes (Require Login)

	user := r.Group("/api/user").Use(middlewares.AuthMiddleware())
	{
		user.POST("/book", controllers.BookTickets(config.DB))
		user.GET("/mybookings", controllers.GetUserBookings(config.DB))
	}

	// Admin Routes (Require Admin Access)

	admin := r.Group("/api/admin")
	admin.Use(middlewares.AdminMiddleware())
	{
		admin.GET("/movies", controllers.AdminListMovies(config.DB))
		admin.POST("/movies", controllers.AdminAddMovie(config.DB))
		admin.DELETE("/movies/:id", controllers.AdminDeleteMovie(config.DB))
		admin.GET("/shows", controllers.AdminListShows(config.DB))
		admin.POST("/shows", controllers.AdminAddShow(config.DB))
		admin.DELETE("/shows/:id", controllers.AdminDeleteShow(config.DB))
		admin.GET("/bookings", controllers.AdminListBookings(config.DB))
		admin.PATCH("/bookings/:id", controllers.AdminUpdateBookingStatus(config.DB))
		admin.GET("/dashboard", controllers.AdminDashboard(config.DB))
	}

	// Public HTML Pages

	r.GET("/", func(c *gin.Context) {
		c.HTML(200, "public_movies.html", gin.H{})
	})

	r.GET("/login", func(c *gin.Context) {
		c.HTML(200, "login.html", gin.H{})
	})

	r.GET("/register", func(c *gin.Context) {
		c.HTML(200, "register.html", gin.H{})
	})

	r.GET("/movie/:id", func(c *gin.Context) {
		c.HTML(200, "public_movie_details.html", gin.H{
			"movie_id": c.Param("id"),
		})
	})

	// Admin HTML Page (Protected)

	// r.GET("/admin/dashboard", func(c *gin.Context) {
	// 	c.HTML(200, "admin_dashboard.html", gin.H{})
	// })

	r.GET("/admin/dashboard", controllers.AdminDashboard(config.DB))

	r.GET("/admin/movies", func(c *gin.Context) {
		token := c.Query("token")
		c.HTML(200, "admin_movies_form.html", gin.H{"Token": token})
	})
	r.GET("/admin/shows", func(c *gin.Context) {
		token := c.Query("token")
		c.HTML(200, "admin_shows_form.html", gin.H{"Token": token})
	})
	r.GET("/admin/bookings", func(c *gin.Context) {
		token := c.Query("token")
		c.HTML(200, "admin_bookings.html", gin.H{"Token": token})
	})

	// Fallback for Unknown Routes

	r.NoRoute(func(c *gin.Context) {
		c.JSON(404, gin.H{"error": "page not found"})
	})

	return r
}
