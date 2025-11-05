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
		api.POST("/user/signup", controllers.UserSignupHandler)
		api.POST("/user/login", controllers.UserLoginHandler)
		api.POST("/admin/signup", controllers.AdminRegister)
		api.POST("/admin/login", controllers.AdminLogin)
		api.POST("/forgot-password", controllers.ForgotPasswordHandler)
		api.POST("/reset-password", controllers.ResetPasswordHandler)

		// New refresh token endpoints
		// api.POST("/refresh", controllers.RefreshTokenHandler)
		api.POST("/logout", controllers.LogoutHandler)

		// Public movie routes
		api.GET("/movies", controllers.GetMovies(config.DB))
		api.GET("/movies/:id", controllers.GetMovieDetails(config.DB))
		api.GET("/movies/:id/shows", controllers.GetShowsByMovie(config.DB))
	}

	// Protected User Routes (Require Login)

	user := r.Group("/api/user").Use(middlewares.AuthMiddleware(), middlewares.UserMiddleware())
	{
		user.GET("/movies", controllers.GetAllMovies(config.DB))
		user.GET("/movies/:id", controllers.GetMovieWithShows(config.DB))
		user.GET("/movies/shows/upcoming", controllers.GetUpcomingShows(config.DB))

		user.POST("/bookings", controllers.CreateBooking(config.DB))
		user.GET("/bookings/:id", controllers.GetBookingDetailsUser(config.DB))
		user.GET("/bookings/user", controllers.GetUserBookings(config.DB))
		user.GET("/shows/:id/seats", controllers.GetShowSeats(config.DB))

		user.GET("/wishlist", controllers.GetWishlist(config.DB))
		user.POST("/wishlist", controllers.AddToWishlist(config.DB))
		user.DELETE("/wishlist/:id", controllers.RemoveFromWishlist(config.DB))

		user.POST("/payments/initiate", controllers.InitiatePayment(config.DB))
		user.POST("/payments/mock/confirm/:id", controllers.MockConfirmPayment(config.DB))
		user.GET("/payments/user", controllers.GetUserPayments(config.DB))

	}

	// Admin Routes (Require Admin Access)

	admin := r.Group("/api/admin")
	admin.Use(middlewares.AdminMiddleware(), middlewares.AuthMiddleware())
	{
		db := config.DB
		analyticsController := controllers.AnalyticsController{DB: db}

		admin.GET("/verify", middlewares.AdminMiddleware(), controllers.AdminVerify(db))

		admin.GET("/dashboard", controllers.AdminDashboard(db))
		admin.GET("/analytics/stats", analyticsController.GetDashboardStats)
		admin.GET("/analytics/daily-revenue", analyticsController.GetDailyRevenue)
		admin.GET("/analytics/bookings-per-movie", analyticsController.GetBookingsPerMovie)
		admin.GET("/analytics/user-activity", analyticsController.GetUserActivity)

		admin.GET("/movies", controllers.AdminListMovies(db))
		admin.POST("/movies", controllers.AdminAddMovie(db))
		admin.DELETE("/movies/:id", controllers.AdminDeleteMovie(db))

		admin.GET("/shows", controllers.AdminListShows(db))
		admin.POST("/shows", controllers.AdminAddShow(db))
		admin.PUT("/shows/:id", controllers.AdminEditShow(db))
		admin.DELETE("/shows/:id", controllers.AdminDeleteShow(db))

		admin.GET("/bookings", controllers.GetAllBookings(db))
		admin.GET("/bookings/:id", controllers.GetBookingDetails(db))
		admin.PUT("/bookings/:id/status", controllers.UpdateBookingStatus(db))
		admin.DELETE("/bookings/:id", controllers.DeleteBooking(db))

		admin.GET("/users", controllers.GetAllUsers(db))
		admin.PUT("/users/:id/block", controllers.BlockUser(db))
		admin.DELETE("/users/:id", controllers.DeleteUser(db))
	}

	// Public HTML Pages

	r.GET("/", func(c *gin.Context) {
		c.HTML(200, "public_movies.html", gin.H{})
	})

	r.GET("/admin/login", middlewares.PreventLoginWhenAuthenticated(), func(c *gin.Context) {
		c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
		c.Header("Pragma", "no-cache")
		c.Header("Expires", "0")
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
	r.GET("/admin/users", func(ctx *gin.Context) {
		token := ctx.Query("token")
		ctx.HTML(200, "admin_users.html", gin.H{"Token": token})
	})

	// Fallback for Unknown Routes

	r.NoRoute(func(c *gin.Context) {
		c.JSON(404, gin.H{"error": "page not found"})
	})

	return r
}
