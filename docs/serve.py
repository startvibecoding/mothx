#!/usr/bin/env python3
"""
Simple HTTP server for VibeCoding documentation.
Run this script from the docs/ directory to serve the documentation.
"""

import http.server
import socketserver
import os

PORT = 8000

class MyHandler(http.server.SimpleHTTPRequestHandler):
    def end_headers(self):
        # Add CORS headers for local development
        self.send_header('Access-Control-Allow-Origin', '*')
        super().end_headers()

if __name__ == "__main__":
    os.chdir(os.path.dirname(os.path.abspath(__file__)))
    
    with socketserver.TCPServer(("", PORT), MyHandler) as httpd:
        print(f"Serving VibeCoding documentation at http://localhost:{PORT}")
        print(f"Open http://localhost:{PORT}/index.html in your browser")
        print("Press Ctrl+C to stop the server")
        try:
            httpd.serve_forever()
        except KeyboardInterrupt:
            print("\nServer stopped.")