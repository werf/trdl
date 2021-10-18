#!/bin/bash -e

VAULT_ADDR="${VAULT_ADDR:-http://127.0.0.1:8200}"
VAULT_TOKEN="${VAULT_TOKEN:-root}"

PROJECT_NAME="${PROJECT_NAME:?ERROR: \$PROJECT_NAME variable required!}"

echo "# VAULT_ADDR=$VAULT_ADDR"
echo "# VAULT_TOKEN=$VAULT_TOKEN"
echo "# PROJECT_NAME=$PROJECT_NAME"

out=$(curl -sS -q -X PUT -H "X-Vault-Token: $VAULT_TOKEN" -H "X-Vault-Request: true" -d "{\"git_tag\":\"${GIT_TAG}\"}" $VAULT_ADDR/v1/$PROJECT_NAME/publish || (echo "Error: unable to start publish operation">&2 && exit 1))
echo "${out}" | jq -e .errors > /dev/null && echo -e "Error: publish operation request failed:\n${out}">&2 && exit 1

taskUUID=$(echo "${out}" | jq -r .data.task_uuid)

echo "# Started task $taskUUID"

while true ; do
	out=$(curl -sS -q -X GET -H "X-Vault-Token: $VAULT_TOKEN" -H "X-Vault-Request: true" $VAULT_ADDR/v1/$PROJECT_NAME/task/$taskUUID || (echo "Error: unable to get task $taskUUID status">&2 && exit 1))
	echo "${out}" | jq -e .errors > /dev/null && echo -e "Error: task $taskUUID status request failed:\n${out}">&2 && exit 1

	taskStatus=$(echo "${out}" | jq -r .data.status)

	if test "x$taskStatus" == "xFAILED" ; then
		failedReason=$(echo "${out}" | jq -r .data.reason)

		out=$(curl -sS -q -X GET -H "X-Vault-Token: $VAULT_TOKEN" -H "X-Vault-Request: true" "$VAULT_ADDR/v1/$PROJECT_NAME/task/$taskUUID/log?limit=0" || (echo "Error: unable to get task $taskUUID logs">&2 && exit 1))
		echo "${out}" | jq -e .errors > /dev/null && echo -e "Error: task $taskUUID status request failed:\n${out}">&2 && exit 1

		taskLogs=$(echo "${out}" | jq -r .data.result)
		echo
		echo -ne "${taskLogs}"

		echo >&2
		echo "Error: task $taskUUID failed: $failedReason">&2
		exit 1
	fi

	if test "x$taskStatus" == "xSUCCEEDED" ; then
		out=$(curl -sS -q -X GET -H "X-Vault-Token: $VAULT_TOKEN" -H "X-Vault-Request: true" "$VAULT_ADDR/v1/$PROJECT_NAME/task/$taskUUID/log?limit=0" || (echo "Error: unable to get task $taskUUID logs">&2 && exit 1))
		echo "${out}" | jq -e .errors > /dev/null && echo -e "Error: task $taskUUID status request failed:\n${out}">&2 && exit 1

		taskLogs=$(echo "${out}" | jq -r .data.result)
		echo
		echo -e "${taskLogs}"

		echo "# Task $taskUUID succeeded"

		break
	fi

	printf "."
	sleep 0.2
done

