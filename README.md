# xfinity-usage
Xfinity internet usage to MQTT via the mobile api.

# Refresh Token
In order to get your refresh token you need to run the Xfinity app on an Android emulator version 24+, login with your credentials and then extract the refresh token from the device.

Example:
```sh
adb root
adb pull /data/data/com.xfinity.digitalhome/shared_prefs/ACCESS_TOKEN_STORE.xml token.xml

# Assuming `xmllint` and `jq` are installed.
{
    ALL="$(xmllint --xpath '//string[@name="ACCESS_TOKEN_STORE_KEY"]/text()' token.xml)"
    REFRESH_TOKEN="$(jq -r .refreshToken <<< $ALL)"
    CLIENT_SECRET="$(jq -r .mLastTokenResponse.request.additionalParameters.client_secret <<< $ALL)"
    echo -e "\nRefresh Token: ${REFRESH_TOKEN}"
    echo -e "\nClient Secret: ${CLIENT_SECRET}\n"
}
```

# Kubernetes Example
This example runs a CronJob every 30m.

```yaml
--
kind: Secret
apiVersion: v1
metadata:
  name: xfinity-usage-secret
  namespace: tools
type: Opaque
data:
  CLIENT_SECRET: (SNIP)
  REFRESH_TOKEN: (SNIP)
  MQTT_USERNAME: (SNIP)
  MQTT_PASSWORD: (SNIP)
--
apiVersion: batch/v1
kind: CronJob
metadata:
  name: xfinity-usage
  namespace: tools
spec:
  schedule: "0/30 * * * *"
  concurrencyPolicy: Replace
  failedJobsHistoryLimit: 3
  successfulJobsHistoryLimit: 1
  startingDeadlineSeconds: 60 # 1 min
  jobTemplate:
    spec:
      backoffLimit: 0
      template:
        spec:
          containers:
            - name: xfinity-usage
              image: ghcr.io/csobrinho/xfinity-usage:main
              imagePullPolicy: IfNotPresent
              resources:
                limits:
                  memory: 256Mi
                  cpu: 200m
                requests:
                  memory: 128Mi
                  cpu: 100m
              env:
                - name: TZ
                  value: America/Los_Angeles
                - name: MQTT_URL
                  value: "mqtt://mosquitto.mosquitto.svc.cluster.local:1883"
              envFrom:
                - secretRef:
                    name: xfinity-usage-secret
              securityContext:
                allowPrivilegeEscalation: false
                readOnlyRootFilesystem: true
                capabilities:
                  drop:
                    - ALL
          restartPolicy: Never
          securityContext:
            runAsUser: 1000
            runAsGroup: 1000
            runAsNonRoot: true
```