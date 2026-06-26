import os
import json
import urllib.request
import urllib.error
import argparse

def send_request(url, api_key, app_key, payload):
    req = urllib.request.Request(
        url,
        data=json.dumps(payload).encode('utf-8'),
        headers={
            "Content-Type": "application/json",
            "DD-API-KEY": api_key,
            "DD-APPLICATION-KEY": app_key
        },
        method="POST"
    )
    try:
        with urllib.request.urlopen(req) as response:
            return json.loads(response.read().decode('utf-8')), None
    except urllib.error.HTTPError as e:
        try:
            err_body = e.read().decode('utf-8')
            return None, f"HTTP Error {e.code}: {err_body}"
        except:
            return None, f"HTTP Error {e.code}: {e.reason}"
    except Exception as e:
        return None, str(e)

def main():
    parser = argparse.ArgumentParser(description="Deploy Datadog Dashboards and Monitors.")
    parser.add_argument("--api-key", help="Datadog API Key")
    parser.add_argument("--app-key", help="Datadog Application Key")
    parser.add_argument("--site", default="datadoghq.com", help="Datadog Site (default: datadoghq.com)")
    args = parser.parse_args()

    # Parse arguments first
    api_key = args.api_key or os.environ.get("DD_API_KEY")
    app_key = args.app_key or os.environ.get("DD_APP_KEY")

    if not api_key:
        api_key = input("Enter Datadog API Key: ").strip()
    if not app_key:
        app_key = input("Enter Datadog Application Key: ").strip()

    if not api_key or not app_key:
        print("Error: Both API Key and Application Key are required.")
        return

    # Base URL based on site
    base_url = f"https://api.{args.site}"

    # Deploy Dashboard
    dashboard_file = os.path.join(os.path.dirname(__file__), "dashboard.json")
    if os.path.exists(dashboard_file):
        print(f"Reading dashboard configuration from {dashboard_file}...")
        with open(dashboard_file, "r") as f:
            dash_payload = json.load(f)
        
        # Remove id from root if it exists because DD API autogenerates it
        if "id" in dash_payload:
            del dash_payload["id"]
        # Remove widget ids
        for w in dash_payload.get("widgets", []):
            if "id" in w:
                del w["id"]

        print("Deploying Dashboard to Datadog...")
        url = f"{base_url}/api/v1/dashboard"
        res, err = send_request(url, api_key, app_key, dash_payload)
        if err:
            print("Failed to deploy Dashboard:", err)
        else:
            print(f"Successfully deployed Dashboard! Title: '{res.get('title')}', ID: {res.get('id')}")
            print(f"URL: https://app.{args.site}/dashboard/{res.get('id')}")
    else:
        print(f"Warning: Dashboard config file not found at {dashboard_file}")

    # Deploy Monitors
    monitors_file = os.path.join(os.path.dirname(__file__), "monitors.json")
    if os.path.exists(monitors_file):
        print(f"\nReading monitors configuration from {monitors_file}...")
        with open(monitors_file, "r") as f:
            monitors_payload = json.load(f)
            
        if not isinstance(monitors_payload, list):
            monitors_payload = [monitors_payload]

        url = f"{base_url}/api/v1/monitor"
        for monitor in monitors_payload:
            print(f"Deploying Monitor '{monitor.get('name')}' to Datadog...")
            res, err = send_request(url, api_key, app_key, monitor)
            if err:
                print(f"Failed to deploy Monitor '{monitor.get('name')}':", err)
            else:
                print(f"Successfully deployed Monitor! Name: '{res.get('name')}', ID: {res.get('id')}")
    else:
        print(f"Warning: Monitors config file not found at {monitors_file}")

if __name__ == "__main__":
    main()
