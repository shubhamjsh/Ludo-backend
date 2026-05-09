# Deployment Guide

## Deploy to Render.com

### Prerequisites
- GitHub account
- Render.com account (free tier available)
- Code pushed to GitHub repository

### Quick Deploy Steps

1. **Push code to GitHub:**
   ```bash
   git add .
   git commit -m "Ready for deployment"
   git push origin main
   ```

2. **Go to Render Dashboard:**
   - Visit: https://dashboard.render.com/

3. **Create PostgreSQL Database:**
   - Click "New +" → "PostgreSQL"
   - Name: `ludo-postgres`
   - Database: `ludo_db`
   - Plan: Free (or Starter for production)
   - Click "Create Database"
   - Save the connection details

4. **Create Web Service:**
   - Click "New +" → "Web Service"
   - Connect your GitHub repo
   - Select `ludo-backend` repository
   - Configure:
     - Name: `ludo-api`
     - Runtime: **Docker**
     - Branch: `main`
     - Plan: Free or Starter ($7/month)

5. **Add Environment Variables:**
   ```
   SERVER_PORT=8080
   JWT_SECRET=<generate-strong-random-string>
   DB_HOST=<postgres-internal-host>
   DB_PORT=5432
   DB_USER=ludo_user
   DB_PASSWORD=<postgres-password>
   DB_NAME=ludo_db
   DB_SSLMODE=require
   ```

6. **Set Health Check:**
   - Path: `/health`

7. **Deploy:**
   - Click "Create Web Service"
   - Wait for build to complete (~3-5 minutes)

### Your API will be live at:
```
https://ludo-api-<random>.onrender.com
```

### Test Deployment
```bash
curl https://your-app.onrender.com/health
curl https://your-app.onrender.com/swagger/index.html
```

### Important Notes

**Free Tier Limitations:**
- App sleeps after 15 minutes of inactivity
- First request after sleep takes ~30 seconds (cold start)
- 750 hours/month free
- Database: 1GB storage, 97 hours/month

**Paid Tier Benefits ($7/month):**
- No sleep
- Instant responses
- Better performance
- More resources

### Monitoring

View logs in Render dashboard:
- Click on your service → "Logs" tab
- See real-time deployment and runtime logs

### Updating Your App

Just push to GitHub:
```bash
git add .
git commit -m "Update feature"
git push origin main
```

Render auto-deploys on every push!

### Troubleshooting

**Build fails:**
- Check Dockerfile syntax
- Verify go.mod and go.sum are committed
- Check build logs in Render dashboard

**Database connection fails:**
- Verify DB_SSLMODE=require
- Check database is in same region
- Verify environment variables are correct

**App crashes on start:**
- Check logs for errors
- Verify all required env vars are set
- Test Docker build locally first

### Local Docker Testing

Before deploying:
```bash
# Build
docker build -t ludo-api:test .

# Run locally
docker run -p 8080:8080 \
  -e DB_HOST=host.docker.internal \
  -e DB_PORT=5432 \
  -e DB_USER=postgres \
  -e DB_PASSWORD=postgres \
  -e DB_NAME=ludo_db \
  -e DB_SSLMODE=disable \
  -e JWT_SECRET=test-secret \
  ludo-api:test

# Test
curl http://localhost:8080/health
```

### Support

- Render Docs: https://render.com/docs
- GitHub Issues: <your-repo-url>/issues
