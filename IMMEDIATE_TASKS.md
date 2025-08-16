# Immediate Release Tasks for v0.4.0

## 1. ✅ Homebrew Tap (Ready to Publish)
**Action Required**: Create repository on GitHub
```bash
# 1. Go to: https://github.com/new
# 2. Create repository: "homebrew-nself" (public)
# 3. Then run:
cd ~/homebrew-nself
git push -u origin main

# Users can then install:
brew tap acamarata/nself
brew install nself
```

## 2. Docker Hub
```bash
cd /Users/admin/Sites/nself
./.releases/docker/build-and-push.sh 0.3.7
docker login -u acamarata
docker push acamarata/nself:0.3.7
docker push acamarata/nself:latest
```

## 3. GitHub Release with Checksums
```bash
cd /Users/admin/Sites/nself
# Create tarball
git archive --format=tar.gz --prefix=nself-0.3.7/ v0.3.7 > nself-0.3.7.tar.gz
# Generate checksums
shasum -a 256 nself-0.3.7.tar.gz > nself-0.3.7.tar.gz.sha256
# Create release on GitHub with these files
```