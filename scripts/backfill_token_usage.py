#!/usr/bin/env python3
"""Backfill New API daily token usage without storing credentials on disk."""

from __future__ import annotations

import getpass
import json
import os
import sys
import time
import urllib.error
import urllib.parse
import urllib.request
from datetime import date, datetime, timedelta, timezone
from typing import Any


DEFAULT_BASE_URL = "https://www.x-x.pro"
DEFAULT_FROM_DATE = "2026-07-01"
DEFAULT_USER_ID = "2"
MAX_BATCH_DAYS = 31
SHANGHAI_TIMEZONE = timezone(timedelta(hours=8))


def api_request(
    base_url: str,
    access_token: str,
    method: str,
    path: str,
    payload: dict[str, Any] | None = None,
) -> dict[str, Any]:
    body = None
    headers = {
        "Accept": "application/json",
        "Authorization": f"Bearer {access_token}",
    }
    if payload is not None:
        body = json.dumps(payload, separators=(",", ":")).encode("utf-8")
        headers["Content-Type"] = "application/json"

    request = urllib.request.Request(
        f"{base_url.rstrip('/')}{path}",
        data=body,
        headers=headers,
        method=method,
    )
    try:
        with urllib.request.urlopen(request, timeout=30) as response:
            response_body = response.read().decode("utf-8")
    except urllib.error.HTTPError as error:
        response_body = error.read().decode("utf-8", errors="replace")
        raise RuntimeError(
            f"HTTP {error.code}: {response_body or error.reason}"
        ) from error
    except urllib.error.URLError as error:
        raise RuntimeError(f"请求失败: {error.reason}") from error

    try:
        result = json.loads(response_body)
    except json.JSONDecodeError as error:
        raise RuntimeError(f"服务器返回的不是 JSON: {response_body[:300]}") from error
    if not isinstance(result, dict):
        raise RuntimeError("服务器返回格式异常")
    return result


def require_success(response: dict[str, Any], action: str) -> Any:
    if response.get("success") is not True:
        message = response.get("message") or "未知错误"
        raise RuntimeError(f"{action}失败: {message}")
    return response.get("data")


def parse_date(value: str, name: str) -> date:
    try:
        return date.fromisoformat(value)
    except ValueError as error:
        raise RuntimeError(f"{name}必须是 YYYY-MM-DD 格式: {value}") from error


def get_active_task(base_url: str, access_token: str) -> dict[str, Any] | None:
    query = urllib.parse.urlencode({"type": "token_usage_daily"})
    response = api_request(
        base_url,
        access_token,
        "GET",
        f"/api/system-task/current?{query}",
    )
    data = require_success(response, "查询当前任务")
    return data if isinstance(data, dict) else None


def wait_for_task(
    base_url: str,
    access_token: str,
    task_id: str,
    poll_seconds: int,
) -> dict[str, Any]:
    last_status = ""
    while True:
        response = api_request(
            base_url,
            access_token,
            "GET",
            f"/api/system-task/{urllib.parse.quote(task_id, safe='')}",
        )
        task = require_success(response, "查询任务状态")
        if not isinstance(task, dict):
            raise RuntimeError("任务状态返回格式异常")

        status = str(task.get("status", ""))
        if status != last_status:
            print(f"  任务 {task_id}: {status}", flush=True)
            last_status = status

        if status == "succeeded":
            return task
        if status == "failed":
            raise RuntimeError(f"任务失败: {task.get('error') or '未提供错误信息'}")
        if status not in {"pending", "running"}:
            raise RuntimeError(f"未知任务状态: {status or '空'}")
        time.sleep(poll_seconds)


def build_batches(from_date: date, to_date: date) -> list[tuple[date, date]]:
    batches: list[tuple[date, date]] = []
    current = from_date
    while current <= to_date:
        batch_end = min(current + timedelta(days=MAX_BATCH_DAYS - 1), to_date)
        batches.append((current, batch_end))
        current = batch_end + timedelta(days=1)
    return batches


def main() -> int:
    base_url = os.environ.get("NEWAPI_URL", DEFAULT_BASE_URL).strip()
    from_date = parse_date(
        os.environ.get("TOKEN_USAGE_FROM_DATE", DEFAULT_FROM_DATE), "开始日期"
    )
    yesterday = datetime.now(SHANGHAI_TIMEZONE).date() - timedelta(days=1)
    to_date = parse_date(
        os.environ.get("TOKEN_USAGE_TO_DATE", yesterday.isoformat()), "结束日期"
    )
    user_setting = os.environ.get("TOKEN_USAGE_USER_ID", DEFAULT_USER_ID).strip()
    poll_seconds = int(os.environ.get("TOKEN_USAGE_POLL_SECONDS", "10"))

    if not base_url.startswith(("https://", "http://")):
        raise RuntimeError("NEWAPI_URL 必须以 http:// 或 https:// 开头")
    if to_date > yesterday:
        raise RuntimeError(f"结束日期不能晚于北京时间昨天 {yesterday}")
    if from_date > to_date:
        raise RuntimeError("开始日期不能晚于结束日期")
    if poll_seconds < 2:
        raise RuntimeError("TOKEN_USAGE_POLL_SECONDS 不能小于 2")

    payload_user_id: int | None
    if user_setting.lower() == "all":
        payload_user_id = None
        scope = "全部用户"
    else:
        try:
            payload_user_id = int(user_setting)
        except ValueError as error:
            raise RuntimeError("TOKEN_USAGE_USER_ID 必须是正整数或 all") from error
        if payload_user_id <= 0:
            raise RuntimeError("TOKEN_USAGE_USER_ID 必须是正整数或 all")
        scope = f"用户 ID {payload_user_id}"

    access_token = os.environ.get("NEWAPI_ROOT_TOKEN", "")
    if not access_token:
        access_token = getpass.getpass("请输入 New API Root 访问令牌: ").strip()
    if not access_token:
        raise RuntimeError("访问令牌不能为空")

    batches = build_batches(from_date, to_date)
    print("\n即将执行历史用量回填：")
    print(f"  地址: {base_url}")
    print(f"  用户: {scope}")
    print(f"  日期: {from_date} 至 {to_date}")
    print(f"  批次: {len(batches)}（每批最多 {MAX_BATCH_DAYS} 天）")

    if os.environ.get("TOKEN_USAGE_ASSUME_YES") != "1":
        confirmation = input("确认开始？请输入 yes: ").strip().lower()
        if confirmation != "yes":
            print("已取消，未创建任何任务。")
            return 0

    active_task = get_active_task(base_url, access_token)
    if active_task is not None:
        raise RuntimeError(
            "已有 token_usage_daily 任务正在运行，"
            f"task_id={active_task.get('task_id')}，请等待完成后重试"
        )

    for index, (batch_from, batch_to) in enumerate(batches, start=1):
        print(
            f"\n[{index}/{len(batches)}] 提交 {batch_from} 至 {batch_to}",
            flush=True,
        )
        payload: dict[str, Any] = {
            "from": batch_from.isoformat(),
            "to": batch_to.isoformat(),
        }
        if payload_user_id is not None:
            payload["user_id"] = payload_user_id

        response = api_request(
            base_url,
            access_token,
            "POST",
            "/api/system-task/token-usage-daily/backfill",
            payload,
        )
        data = require_success(response, "创建回填任务")
        if not isinstance(data, dict) or not isinstance(data.get("task"), dict):
            raise RuntimeError("创建任务返回格式异常")
        if data.get("created") is not True:
            task_id = data["task"].get("task_id")
            raise RuntimeError(f"未创建新任务，当前活动任务为 {task_id}")

        task_id = str(data["task"].get("task_id", ""))
        if not task_id:
            raise RuntimeError("创建任务成功，但未返回 task_id")
        task = wait_for_task(base_url, access_token, task_id, poll_seconds)
        result = task.get("result")
        print(
            "  本批完成: "
            + json.dumps(result, ensure_ascii=False, separators=(",", ":")),
            flush=True,
        )

    print("\n全部历史用量回填完成。")
    return 0


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except KeyboardInterrupt:
        print("\n已中断。正在运行的服务端任务不会被取消。", file=sys.stderr)
        raise SystemExit(130)
    except Exception as error:
        print(f"\n错误: {error}", file=sys.stderr)
        raise SystemExit(1)
