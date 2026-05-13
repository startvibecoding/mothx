# VibeCoding Documentation

This directory contains the documentation for VibeCoding in both Chinese and English.

## Structure

```
docs/
├── index.html          # HTML documentation with language switching
├── serve.py            # Simple HTTP server for local development
├── en/                 # English documentation
│   ├── README.md
│   ├── getting-started.md
│   ├── cli-reference.md
│   ├── configuration.md
│   ├── architecture.md
│   ├── tools.md
│   ├── security.md
│   ├── sessions.md
│   ├── development.md
│   └── faq.md
└── zh/                 # Chinese documentation
    ├── README.md
    ├── getting-started.md
    ├── cli-reference.md
    ├── configuration.md
    ├── architecture.md
    ├── tools.md
    ├── security.md
    ├── sessions.md
    ├── development.md
    └── faq.md
```

## Viewing Documentation

### Option 1: Open HTML file directly

Open `index.html` in a modern web browser. Note that some browsers may block local file access due to CORS policies.

### Option 2: Use the Python server (Recommended)

1. Navigate to the `docs` directory:
   ```bash
   cd docs
   ```

2. Run the server:
   ```bash
   python3 serve.py
   ```

3. Open your browser and go to:
   ```
   http://localhost:8000/index.html
   ```

### Option 3: Use any HTTP server

You can use any HTTP server to serve the documentation directory. For example:

```bash
# Using Python's built-in server
cd docs
python3 -m http.server 8000

# Using Node.js's http-server (if installed)
cd docs
npx http-server
```

## Language Switching

The HTML documentation supports switching between Chinese and English:
- Click the "中文" button to switch to Chinese
- Click the "English" button to switch to English

The documentation will remember your language preference during the session.

## Adding New Documentation

1. Create the document in both `en/` and `zh/` directories
2. Update the `docs` object in `index.html` to include the new document
3. Update the sidebar navigation in the HTML file

## Building for Production

For production deployment, you may want to:
1. Minify the HTML file
2. Bundle the JavaScript
3. Use a proper web server like Nginx or Apache
4. Set up proper caching headers

## License

The documentation is licensed under the same license as the VibeCoding project.