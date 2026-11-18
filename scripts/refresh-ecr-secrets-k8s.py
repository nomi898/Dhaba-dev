import base64
import json
import os
import ssl
import subprocess
import urllib.request

REGION = "eu-north-1"
ACCOUNT_ID = "270254976774"
REGISTRY = f"{ACCOUNT_ID}.dkr.ecr.{REGION}.amazonaws.com"

TOKEN_PATH = "/var/run/secrets/kubernetes.io/serviceaccount/token"
CA_PATH = "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt"

with open(TOKEN_PATH) as f:
    token = f.read().strip()

context = ssl.create_default_context(cafile=CA_PATH)

password = subprocess.run(
    ["aws", "ecr", "get-login-password", "--region", REGION],
    check=True,
    capture_output=True,
    text=True
).stdout.strip()

auth = base64.b64encode(
    f"AWS:{password}".encode()
).decode()

docker_config = {
    "auths": {
        REGISTRY: {
            "username": "AWS",
            "password": password,
            "auth": auth
        }
    }
}

encoded_config = base64.b64encode(
    json.dumps(docker_config).encode()
).decode()

patch = json.dumps({
    "type": "kubernetes.io/dockerconfigjson",
    "data": {
        ".dockerconfigjson": encoded_config
    }
}).encode()

for namespace in ["staging", "production"]:
    url = (
        "https://kubernetes.default.svc"
        f"/api/v1/namespaces/{namespace}/secrets/ecr-pull-secret"
    )

    request = urllib.request.Request(
        url,
        data=patch,
        method="PATCH",
        headers={
            "Authorization": f"Bearer {token}",
            "Content-Type": "application/merge-patch+json"
        }
    )

    with urllib.request.urlopen(
        request, context=context, timeout=20
    ) as response:
        print(f"{namespace}: refreshed successfully ({response.status})")

print("ECR credentials refreshed in both namespaces.")