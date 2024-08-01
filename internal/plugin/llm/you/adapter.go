package you

import (
	"chatgpt-adapter/internal/common"
	"chatgpt-adapter/internal/gin.handler/response"
	"chatgpt-adapter/internal/plugin"
	"chatgpt-adapter/internal/vars"
	"chatgpt-adapter/logger"
	"chatgpt-adapter/pkg"
	"context"
	"errors"
	"fmt"
	"github.com/bincooo/emit.io"
	"github.com/bincooo/you.com"
	"github.com/gin-gonic/gin"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

const iCookie = "stytch_csrf_private_token=NmCbmAUMJA39wXZzImilAX6xRvYBHsxrQO6ESbow_2cK; ydc_stytch_session_jwt=eyJhbGciOiJSUzI1NiIsImtpZCI6Imp3ay1saXZlLTIwYzU3NzYzLTcxYzYtNDdmNC1hMWVmLTQ2NTZlMTU5ZTJlZSIsInR5cCI6IkpXVCJ9.eyJhdWQiOlsicHJvamVjdC1saXZlLTlkZWE3ZGI1LTJlMTUtNGE3ZC1iYjFmLTJjYjg0ODBlYTliMCJdLCJhdXRoMF9pZCI6bnVsbCwiZXhwIjoxNzIxOTI0NTg1LCJodHRwczovL3N0eXRjaC5jb20vc2Vzc2lvbiI6eyJpZCI6InNlc3Npb24tbGl2ZS1lOWY2OGUxYy05NzdlLTRkZWUtYTVjMC1iYzQzZGNmMjM5MDAiLCJzdGFydGVkX2F0IjoiMjAyNC0wNy0yNVQxNjoxODowNVoiLCJsYXN0X2FjY2Vzc2VkX2F0IjoiMjAyNC0wNy0yNVQxNjoxODowNVoiLCJleHBpcmVzX2F0IjoiMjAyNC0xMC0yM1QxNjoxODowNVoiLCJhdHRyaWJ1dGVzIjp7InVzZXJfYWdlbnQiOiJNb3ppbGxhLzUuMCAoTWFjaW50b3NoOyBJbnRlbCBNYWMgT1MgWCAxMF8xNV83KSBBcHBsZVdlYktpdC81MzcuMzYgKEtIVE1MLCBsaWtlIEdlY2tvKSBDaHJvbWUvMTI1LjAuMC4wIFNhZmFyaS81MzcuMzYgRWRnLzEyNS4wLjAuMCIsImlwX2FkZHJlc3MiOiIxMDcuMTY3LjE4Ljk5In0sImF1dGhlbnRpY2F0aW9uX2ZhY3RvcnMiOlt7InR5cGUiOiJvdHAiLCJkZWxpdmVyeV9tZXRob2QiOiJlbWFpbCIsImxhc3RfYXV0aGVudGljYXRlZF9hdCI6IjIwMjQtMDctMjVUMTY6MTg6MDVaIiwiZW1haWxfZmFjdG9yIjp7ImVtYWlsX2lkIjoiZW1haWwtbGl2ZS0wZDIwYmVlOS04NGNjLTRhNTItYWI0OC00ZDhlNjRkN2Y1ZDgiLCJlbWFpbF9hZGRyZXNzIjoidHJ2dGFnYmRAc2hhcmtsYXNlcnMuY29tIn19XX0sImlhdCI6MTcyMTkyNDI4NSwiaXNzIjoic3R5dGNoLmNvbS9wcm9qZWN0LWxpdmUtOWRlYTdkYjUtMmUxNS00YTdkLWJiMWYtMmNiODQ4MGVhOWIwIiwibmJmIjoxNzIxOTI0Mjg1LCJzdWIiOiJ1c2VyLWxpdmUtMzkzNjY5MjEtNmMyMS00ZjBmLTg4N2MtNzNmMmFiNTJkN2U0In0.bLnP74_ay71y6F2zu6VSpNNdqsK-pH25rt9oi42x9SOxg0se0RSCh7Rl--Dj2L-GsLCl8NMgMao9vC8oQjB_nls5rxJ0AHe3Ar1tneyWjIb4Qx5TyNXY15OpW0DafuNpFmyY24CcpcvhMv5OocRmMUuQZECeB_5Y8ygzRhcUrnDrNP9ZTLjd-nrFY4Y6IhU9AQfwUJi-NoEKfbH9bdUmDQanJdB1f_JH-6cDer7iZ925kGxAd6D09XgyqBiVXFcxHo4f0yyy9QSOqvAuHI_z6ob-0vCn50jDwI9IFosY-in5NkKl40tF7j0Fo22jGco0f8dRYu1_RDWeXb9y7lHm_g; ydc_stytch_session=MC2cFqTR-n4ow91D4jxqT2nVwuBptjhbrOh-5U4_TfFi; ab.storage.userId.dcee0642-d796-4a7b-9e56-a0108e133b07=g%3Auser-live-39366921-6c21-4f0f-887c-73f2ab52d7e4%7Ce%3Aundefined%7Cc%3A1721924285412%7Cl%3A1721924285414; _ga_2N7ZM9C56V=GS1.1.1721924263.1.1.1721924285.0.0.1068401432; stytch_session=MC2cFqTR-n4ow91D4jxqT2nVwuBptjhbrOh-5U4_TfFi; _gtmeec=eyJjdCI6IjdlOGVlYTVjYzYwOTgwMjcwYzljZWI3NWNlOGMwODdkNDhkNzI2MTEwZmQzZDE3OTIxZjc3NGVlZmQ4ZTE4ZDgiLCJzdCI6IjY5NTkwOTcwMDFkMTA1MDFhYzdkNTRjMGJkYjhkYjYxNDIwZjY1OGYyOTIyY2MyNmU0NmQ1MzYxMTlhMzExMjYiLCJ6cCI6IjQ2MWFkOWFjYjBjYTFiMTUxMmFiZjU4Njk1YTZiNTRkZmNhZjg1MWIzMTg1Mzc2Njk0MDFmNzY2NWNlOTJiZTUiLCJjb3VudHJ5IjoiNzlhZGIyYTJmY2U1YzZiYTIxNWZlNWYyN2Y1MzJkNGU3ZWRiYWM0YjZhNWUwOWUxZWYzYTA4MDg0YTkwNDYyMSJ9; _clsk=c02txe%7C1721924264191%7C1%7C1%7Cv.clarity.ms%2Fcollect; FPAU=1.2.659459949.1721924264; daily_query_date=Fri%20Jul%2026%202024; _ga=GA1.1.1353432611.1721924264; AF_DEFAULT_MEASUREMENT_STATUS=true; ab.storage.deviceId.dcee0642-d796-4a7b-9e56-a0108e133b07=g%3A5fdf5fe0-57c2-0d0a-139b-2437cb9cdd1f%7Ce%3Aundefined%7Cc%3A1721924260749%7Cl%3A1721924285413; _clck=1qljrb3%7C2%7Cfnr%7C0%7C1667; ab.storage.sessionId.dcee0642-d796-4a7b-9e56-a0108e133b07=g%3A5aa146a6-75fd-144c-fd59-95a4b620567d%7Ce%3A1721926085417%7Cc%3A1721924285413%7Cl%3A1721924285417; FPID=FPID2.2.qK1k5Sgtz3UmGzXz1sCpr0AnimQeV1Rz%2F4ag6YRZmLg%3D.1721924264; AF_SYNC=1721924266118; youchat_smart_learn=true; youchat_personalization=true; FPGSID=1.1721924264.1721924264.G-WYGVQX1R23.pG1pq523-vlgv43vjgdJLA; daily_query_count=0; uuid_guest=b8891c9a-bb0d-461d-83cd-c71208f5d327; total_query_count=0; stytch_session_jwt=eyJhbGciOiJSUzI1NiIsImtpZCI6Imp3ay1saXZlLTIwYzU3NzYzLTcxYzYtNDdmNC1hMWVmLTQ2NTZlMTU5ZTJlZSIsInR5cCI6IkpXVCJ9.eyJhdWQiOlsicHJvamVjdC1saXZlLTlkZWE3ZGI1LTJlMTUtNGE3ZC1iYjFmLTJjYjg0ODBlYTliMCJdLCJhdXRoMF9pZCI6bnVsbCwiZXhwIjoxNzIxOTI0NTg1LCJodHRwczovL3N0eXRjaC5jb20vc2Vzc2lvbiI6eyJpZCI6InNlc3Npb24tbGl2ZS1lOWY2OGUxYy05NzdlLTRkZWUtYTVjMC1iYzQzZGNmMjM5MDAiLCJzdGFydGVkX2F0IjoiMjAyNC0wNy0yNVQxNjoxODowNVoiLCJsYXN0X2FjY2Vzc2VkX2F0IjoiMjAyNC0wNy0yNVQxNjoxODowNVoiLCJleHBpcmVzX2F0IjoiMjAyNC0xMC0yM1QxNjoxODowNVoiLCJhdHRyaWJ1dGVzIjp7InVzZXJfYWdlbnQiOiJNb3ppbGxhLzUuMCAoTWFjaW50b3NoOyBJbnRlbCBNYWMgT1MgWCAxMF8xNV83KSBBcHBsZVdlYktpdC81MzcuMzYgKEtIVE1MLCBsaWtlIEdlY2tvKSBDaHJvbWUvMTI1LjAuMC4wIFNhZmFyaS81MzcuMzYgRWRnLzEyNS4wLjAuMCIsImlwX2FkZHJlc3MiOiIxMDcuMTY3LjE4Ljk5In0sImF1dGhlbnRpY2F0aW9uX2ZhY3RvcnMiOlt7InR5cGUiOiJvdHAiLCJkZWxpdmVyeV9tZXRob2QiOiJlbWFpbCIsImxhc3RfYXV0aGVudGljYXRlZF9hdCI6IjIwMjQtMDctMjVUMTY6MTg6MDVaIiwiZW1haWxfZmFjdG9yIjp7ImVtYWlsX2lkIjoiZW1haWwtbGl2ZS0wZDIwYmVlOS04NGNjLTRhNTItYWI0OC00ZDhlNjRkN2Y1ZDgiLCJlbWFpbF9hZGRyZXNzIjoidHJ2dGFnYmRAc2hhcmtsYXNlcnMuY29tIn19XX0sImlhdCI6MTcyMTkyNDI4NSwiaXNzIjoic3R5dGNoLmNvbS9wcm9qZWN0LWxpdmUtOWRlYTdkYjUtMmUxNS00YTdkLWJiMWYtMmNiODQ4MGVhOWIwIiwibmJmIjoxNzIxOTI0Mjg1LCJzdWIiOiJ1c2VyLWxpdmUtMzkzNjY5MjEtNmMyMS00ZjBmLTg4N2MtNzNmMmFiNTJkN2U0In0.bLnP74_ay71y6F2zu6VSpNNdqsK-pH25rt9oi42x9SOxg0se0RSCh7Rl--Dj2L-GsLCl8NMgMao9vC8oQjB_nls5rxJ0AHe3Ar1tneyWjIb4Qx5TyNXY15OpW0DafuNpFmyY24CcpcvhMv5OocRmMUuQZECeB_5Y8ygzRhcUrnDrNP9ZTLjd-nrFY4Y6IhU9AQfwUJi-NoEKfbH9bdUmDQanJdB1f_JH-6cDer7iZ925kGxAd6D09XgyqBiVXFcxHo4f0yyy9QSOqvAuHI_z6ob-0vCn50jDwI9IFosY-in5NkKl40tF7j0Fo22jGco0f8dRYu1_RDWeXb9y7lHm_g; safesearch_guest=Moderate; FPLC=gFa%2FormB8cVVU9LRFHQ7sl5QnFRuisVgelPITOmaiwYew2T4%2BqEvk1H%2B0XOvxuYeIcZrwaz6I0b1ITMxJTdKjOgSZMb5y%2By9gNs7ICApfyj9gcLAxnuYItmE1eggAw%3D%3D; uuid_guest_backup=b8891c9a-bb0d-461d-83cd-c71208f5d327; afUserId=490c9d99-d837-420b-a2e3-fcefbbda947c-p; you_subscription=free; ai_model=gpt_4o; youpro_subscription=false; \n2024-07-29 18:15:21 <internal> common/poll.go:81 | [INFO] [you] PollContainer 冷却完毕: stytch_csrf_private_token=NIyimhqpg7OZ8JS6CsL-3TAErvkvBeVBW-A-36yokNuo; ydc_stytch_session_jwt=eyJhbGciOiJSUzI1NiIsImtpZCI6Imp3ay1saXZlLTIwYzU3NzYzLTcxYzYtNDdmNC1hMWVmLTQ2NTZlMTU5ZTJlZSIsInR5cCI6IkpXVCJ9.eyJhdWQiOlsicHJvamVjdC1saXZlLTlkZWE3ZGI1LTJlMTUtNGE3ZC1iYjFmLTJjYjg0ODBlYTliMCJdLCJhdXRoMF9pZCI6bnVsbCwiZXhwIjoxNzIxOTI0NjIwLCJodHRwczovL3N0eXRjaC5jb20vc2Vzc2lvbiI6eyJpZCI6InNlc3Npb24tbGl2ZS0zZDMxNzM3ZC1jZjMwLTRiNjQtOWIwZC0yMGI4NWQ5YjkwMDciLCJzdGFydGVkX2F0IjoiMjAyNC0wNy0yNVQxNjoxODo0MFoiLCJsYXN0X2FjY2Vzc2VkX2F0IjoiMjAyNC0wNy0yNVQxNjoxODo0MFoiLCJleHBpcmVzX2F0IjoiMjAyNC0xMC0yM1QxNjoxODo0MFoiLCJhdHRyaWJ1dGVzIjp7InVzZXJfYWdlbnQiOiJNb3ppbGxhLzUuMCAoTWFjaW50b3NoOyBJbnRlbCBNYWMgT1MgWCAxMF8xNV83KSBBcHBsZVdlYktpdC81MzcuMzYgKEtIVE1MLCBsaWtlIEdlY2tvKSBDaHJvbWUvMTI1LjAuMC4wIFNhZmFyaS81MzcuMzYgRWRnLzEyNS4wLjAuMCIsImlwX2FkZHJlc3MiOiIxMDcuMTY3LjE4Ljk5In0sImF1dGhlbnRpY2F0aW9uX2ZhY3RvcnMiOlt7InR5cGUiOiJvdHAiLCJkZWxpdmVyeV9tZXRob2QiOiJlbWFpbCIsImxhc3RfYXV0aGVudGljYXRlZF9hdCI6IjIwMjQtMDctMjVUMTY6MTg6NDBaIiwiZW1haWxfZmFjdG9yIjp7ImVtYWlsX2lkIjoiZW1haWwtbGl2ZS04NzYyZDVhYy00MmZjLTQ2NGMtOGYyZi0yNjAwMjdmMzRjZGIiLCJlbWFpbF9hZGRyZXNzIjoicGNram5tb2lAc2hhcmtsYXNlcnMuY29tIn19XX0sImlhdCI6MTcyMTkyNDMyMCwiaXNzIjoic3R5dGNoLmNvbS9wcm9qZWN0LWxpdmUtOWRlYTdkYjUtMmUxNS00YTdkLWJiMWYtMmNiODQ4MGVhOWIwIiwibmJmIjoxNzIxOTI0MzIwLCJzdWIiOiJ1c2VyLWxpdmUtODZiNmY5MzctN2ViOS00NTQyLThkMmUtMWI3ZTYyMmMwOWE0In0.HdHwPUkWdykWHplHFTDztTJU8xcDh8wCKRbCmjVLPFqRgJqVqHV_TSjxVvv1SUe844KvKeVcPnXV9_b3v_HfnxYUaPHss8z76MWlnWvkd4WabZtAt27yBEkDD0pihII2hCEqUYU5Qyrwq7OxXJ_ajn9lT6tRvUv_WBEK_XhPN3qnSdztAjmuf96yOxhlTDnOG-ywsPZS_1Rc6PEm1ZMg5f7rkxfHowRt-G_IOwekaUgUn-ns4MAK3bESUNVvgSKyTkqgMVbg-_m9f9D8gKOaT4qMGjqqyPlC0d6nD8mjHfOb7MH6V6dAiQpyrRzNI__-ZzC5izEsfs8xWpWKrRx5kw; ydc_stytch_session=7h3aEz0FE_5RwhovYrRYFPvbCySGmSrlXZG3vCCQc5jk; ab.storage.userId.dcee0642-d796-4a7b-9e56-a0108e133b07=g%3Auser-live-86b6f937-7eb9-4542-8d2e-1b7e622c09a4%7Ce%3Aundefined%7Cc%3A1721924320490%7Cl%3A1721924320491; _ga_2N7ZM9C56V=GS1.1.1721924297.1.1.1721924320.0.0.1903641966; stytch_session=7h3aEz0FE_5RwhovYrRYFPvbCySGmSrlXZG3vCCQc5jk; afUserId=50e9f2d3-e5ba-4a1c-a8c2-dedfda3778d9-p; _clsk=s7izar%7C1721924298605%7C1%7C1%7Cv.clarity.ms%2Fcollect; AF_DEFAULT_MEASUREMENT_STATUS=true; FPAU=1.2.1637010692.1721924298; daily_query_date=Fri%20Jul%2026%202024; _ga=GA1.1.1217787069.1721924297; ab.storage.deviceId.dcee0642-d796-4a7b-9e56-a0108e133b07=g%3A7db5c59f-84c7-6149-75dd-531910b3dfc3%7Ce%3Aundefined%7Cc%3A1721924295075%7Cl%3A1721924320491; _clck=11qcgun%7C2%7Cfnr%7C0%7C1667; _gtmeec=eyJjdCI6IjdlOGVlYTVjYzYwOTgwMjcwYzljZWI3NWNlOGMwODdkNDhkNzI2MTEwZmQzZDE3OTIxZjc3NGVlZmQ4ZTE4ZDgiLCJzdCI6IjY5NTkwOTcwMDFkMTA1MDFhYzdkNTRjMGJkYjhkYjYxNDIwZjY1OGYyOTIyY2MyNmU0NmQ1MzYxMTlhMzExMjYiLCJ6cCI6IjQ2MWFkOWFjYjBjYTFiMTUxMmFiZjU4Njk1YTZiNTRkZmNhZjg1MWIzMTg1Mzc2Njk0MDFmNzY2NWNlOTJiZTUiLCJjb3VudHJ5IjoiNzlhZGIyYTJmY2U1YzZiYTIxNWZlNWYyN2Y1MzJkNGU3ZWRiYWM0YjZhNWUwOWUxZWYzYTA4MDg0YTkwNDYyMSJ9; ab.storage.sessionId.dcee0642-d796-4a7b-9e56-a0108e133b07=g%3Ac025c9b7-5856-6a7b-2083-c771d96e0d43%7Ce%3A1721926120494%7Cc%3A1721924320490%7Cl%3A1721924320494; FPID=FPID2.2.hmTRIlnB0gY3uBTsFfxOZ7FkhQUSrJ%2FNi8EkbdouU3Y%3D.1721924297; AF_SYNC=1721924299837; youchat_smart_learn=true; youchat_personalization=true; FPGSID=1.1721924297.1721924297.G-WYGVQX1R23.0PURBe75bAQnrQREgxYnBw; daily_query_count=0; uuid_guest=20ec322a-2d34-4c19-bad0-fdf3a02374a9; total_query_count=0; stytch_session_jwt=eyJhbGciOiJSUzI1NiIsImtpZCI6Imp3ay1saXZlLTIwYzU3NzYzLTcxYzYtNDdmNC1hMWVmLTQ2NTZlMTU5ZTJlZSIsInR5cCI6IkpXVCJ9.eyJhdWQiOlsicHJvamVjdC1saXZlLTlkZWE3ZGI1LTJlMTUtNGE3ZC1iYjFmLTJjYjg0ODBlYTliMCJdLCJhdXRoMF9pZCI6bnVsbCwiZXhwIjoxNzIxOTI0NjIwLCJodHRwczovL3N0eXRjaC5jb20vc2Vzc2lvbiI6eyJpZCI6InNlc3Npb24tbGl2ZS0zZDMxNzM3ZC1jZjMwLTRiNjQtOWIwZC0yMGI4NWQ5YjkwMDciLCJzdGFydGVkX2F0IjoiMjAyNC0wNy0yNVQxNjoxODo0MFoiLCJsYXN0X2FjY2Vzc2VkX2F0IjoiMjAyNC0wNy0yNVQxNjoxODo0MFoiLCJleHBpcmVzX2F0IjoiMjAyNC0xMC0yM1QxNjoxODo0MFoiLCJhdHRyaWJ1dGVzIjp7InVzZXJfYWdlbnQiOiJNb3ppbGxhLzUuMCAoTWFjaW50b3NoOyBJbnRlbCBNYWMgT1MgWCAxMF8xNV83KSBBcHBsZVdlYktpdC81MzcuMzYgKEtIVE1MLCBsaWtlIEdlY2tvKSBDaHJvbWUvMTI1LjAuMC4wIFNhZmFyaS81MzcuMzYgRWRnLzEyNS4wLjAuMCIsImlwX2FkZHJlc3MiOiIxMDcuMTY3LjE4Ljk5In0sImF1dGhlbnRpY2F0aW9uX2ZhY3RvcnMiOlt7InR5cGUiOiJvdHAiLCJkZWxpdmVyeV9tZXRob2QiOiJlbWFpbCIsImxhc3RfYXV0aGVudGljYXRlZF9hdCI6IjIwMjQtMDctMjVUMTY6MTg6NDBaIiwiZW1haWxfZmFjdG9yIjp7ImVtYWlsX2lkIjoiZW1haWwtbGl2ZS04NzYyZDVhYy00MmZjLTQ2NGMtOGYyZi0yNjAwMjdmMzRjZGIiLCJlbWFpbF9hZGRyZXNzIjoicGNram5tb2lAc2hhcmtsYXNlcnMuY29tIn19XX0sImlhdCI6MTcyMTkyNDMyMCwiaXNzIjoic3R5dGNoLmNvbS9wcm9qZWN0LWxpdmUtOWRlYTdkYjUtMmUxNS00YTdkLWJiMWYtMmNiODQ4MGVhOWIwIiwibmJmIjoxNzIxOTI0MzIwLCJzdWIiOiJ1c2VyLWxpdmUtODZiNmY5MzctN2ViOS00NTQyLThkMmUtMWI3ZTYyMmMwOWE0In0.HdHwPUkWdykWHplHFTDztTJU8xcDh8wCKRbCmjVLPFqRgJqVqHV_TSjxVvv1SUe844KvKeVcPnXV9_b3v_HfnxYUaPHss8z76MWlnWvkd4WabZtAt27yBEkDD0pihII2hCEqUYU5Qyrwq7OxXJ_ajn9lT6tRvUv_WBEK_XhPN3qnSdztAjmuf96yOxhlTDnOG-ywsPZS_1Rc6PEm1ZMg5f7rkxfHowRt-G_IOwekaUgUn-ns4MAK3bESUNVvgSKyTkqgMVbg-_m9f9D8gKOaT4qMGjqqyPlC0d6nD8mjHfOb7MH6V6dAiQpyrRzNI__-ZzC5izEsfs8xWpWKrRx5kw; safesearch_guest=Moderate; FPLC=1P1%2BlZ%2B1h1nSJLKhE8OeR7Cv%2Fd4s1s4omACOzB0C71s6eMBG%2F%2F1BPglp3kj1u%2BPYDK4qxJEqDH4cQqCuB2jCcTng%2FE3%2FRLaY3NvNqIpBur1MInzRe5gHqfbRNkq8ug%3D%3D; uuid_guest_backup=20ec322a-2d34-4c19-bad0-fdf3a02374a9; you_subscription=free; ai_model=gpt_4o; youpro_subscription=false;"

var (
	mu sync.Mutex

	Adapter = API{}
	Model   = "you"

	lang      = "cn-ZN,cn;q=0.9"
	clearance = ""
	userAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.0 Safari/537.36 Edg/125.0.0.0"

	notice           string
	youRollContainer *common.PollContainer[string]
)

type API struct {
	plugin.BaseAdapter
}

func init() {
	common.AddInitialized(func() {
		cookies := pkg.Config.GetStringSlice("you.cookies")
		if len(cookies) == 0 {
			return
		}

		notice = pkg.Config.GetString("you.notice")
		youRollContainer = common.NewPollContainer[string]("you", cookies, 6*time.Hour)
		youRollContainer.Condition = Condition

		if pkg.Config.GetBool("serverless.enabled") {
			port := pkg.Config.GetString("you.helper")
			if port == "" {
				port = "8081"
			}
			you.Exec(port, vars.Proxies, os.Stdout, os.Stdout)
			common.AddExited(you.Exit)
		}

		go timer()
	})
}

func timer() {
	m30 := 30 * time.Minute

	for {
		time.Sleep(m30)
		if clearance != "" {
			timeout, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			chat := you.New(iCookie, you.GPT_4, vars.Proxies)
			chat.CloudFlare(clearance, userAgent, lang)
			chat.Client(plugin.HTTPClient)
			_, err := chat.State(timeout)
			cancel()
			if err == nil {
				continue
			}

			var se emit.Error
			if !errors.As(err, &se) {
				logger.Error("定时器 you.com 过盾检查失败：%v", err)
				continue
			}

			if se.Code == 403 {
				// 需要重新过盾
				clearance = ""
			} else {
				logger.Error("定时器 you.com 过盾检查失败：%v", err)
				continue
			}
		}

		// 尝试过盾
		if err := tryCloudFlare(); err != nil {
			logger.Errorf("you.com 尝试过盾失败：%v", err)
			continue
		}

		logger.Info("定时器执行 you.com 过盾成功")
	}
}

func (API) Match(_ *gin.Context, model string) bool {
	if strings.HasPrefix(model, "you/") {
		switch model[4:] {
		case you.GPT_4,
			you.GPT_4o,
			you.GPT_4_TURBO,
			you.CLAUDE_2,
			you.CLAUDE_3_HAIKU,
			you.CLAUDE_3_SONNET,
			you.CLAUDE_3_5_SONNET,
			you.CLAUDE_3_OPUS:
			return true
		}
	}
	return false
}

func (API) Models() []plugin.Model {
	return []plugin.Model{
		{
			Id:      "you/" + you.GPT_4,
			Object:  "model",
			Created: 1686935002,
			By:      Model + "-adapter",
		},
		{
			Id:      "you/" + you.GPT_4o,
			Object:  "model",
			Created: 1686935002,
			By:      Model + "-adapter",
		},
		{
			Id:      "you/" + you.GPT_4_TURBO,
			Object:  "model",
			Created: 1686935002,
			By:      Model + "-adapter",
		},
		{
			Id:      "you/" + you.CLAUDE_2,
			Object:  "model",
			Created: 1686935002,
			By:      Model + "-adapter",
		},
		{
			Id:      "you/" + you.CLAUDE_3_HAIKU,
			Object:  "model",
			Created: 1686935002,
			By:      Model + "-adapter",
		},
		{
			Id:      "you/" + you.CLAUDE_3_SONNET,
			Object:  "model",
			Created: 1686935002,
			By:      Model + "-adapter",
		},
		{
			Id:      "you/" + you.CLAUDE_3_5_SONNET,
			Object:  "model",
			Created: 1686935002,
			By:      Model + "-adapter",
		},
		{
			Id:      "you/" + you.CLAUDE_3_OPUS,
			Object:  "model",
			Created: 1686935002,
			By:      Model + "-adapter",
		},
	}
}

func (API) Completion(ctx *gin.Context) {
	var (
		retry   = 3
		cookies []string

		echo = ctx.GetBool(vars.GinEcho)
	)

	defer func() {
		for _, value := range cookies {
			resetMarker(value)
		}
	}()

	var (
		proxies    = ctx.GetString("proxies")
		completion = common.GetGinCompletion(ctx)
		matchers   = common.GetGinMatchers(ctx)
	)

	completion.Model = completion.Model[4:]
	fileMessage, message, tokens, err := mergeMessages(ctx, completion)
	if err != nil {
		logger.Error(err)
		response.Error(ctx, -1, err)
		return
	}

	if echo {
		response.Echo(ctx, completion.Model, fmt.Sprintf("--------FILE MESSAGE--------:\n%s\n\n\n--------CURR QUESTION--------:\n%s", fileMessage, message), completion.Stream)
		return
	}

	if youRollContainer.Len() == 0 {
		response.Error(ctx, -1, "empty cookies")
		return
	}

label:
	retry--
	cookie, err := youRollContainer.Poll()
	if err != nil {
		logger.Error(err)
		response.Error(ctx, -1, err)
		return
	}

	cookies = append(cookies, cookie)
	if plugin.NeedToToolCall(ctx) {
		if completeToolCalls(ctx, cookie, proxies, completion) {
			return
		}
	}

	chat := you.New(cookie, completion.Model, proxies)
	chat.LimitWithE(true)
	chat.Client(plugin.HTTPClient)

	if err = tryCloudFlare(); err != nil {
		response.Error(ctx, -1, err)
		return
	}

	chat.CloudFlare(clearance, userAgent, lang)

	var cancel chan error
	cancel, matchers = joinMatchers(ctx, matchers)
	ctx.Set(ginTokens, tokens)

	ch, err := chat.Reply(common.GetGinContext(ctx), nil, fileMessage, message)
	if err != nil {
		logger.Error(err)
		var se emit.Error
		code := -1
		if errors.As(err, &se) && se.Code > 400 {
			_ = youRollContainer.SetMarker(cookie, 2)
			// 403 重定向？？？
			if se.Code == 403 {
				code = 429
				cleanCf()
			}
		}

		if strings.Contains(err.Error(), "ZERO QUOTA") {
			_ = youRollContainer.SetMarker(cookie, 2)
			code = 429
		}

		if retry > 0 {
			goto label
		}
		response.Error(ctx, code, err)
		return
	}

	content := waitResponse(ctx, matchers, cancel, ch, completion.Stream)
	if content == "" && response.NotResponse(ctx) {
		response.Error(ctx, -1, "EMPTY RESPONSE")
	}
}

func (API) Messages(ctx *gin.Context) {
	var (
		retry   = 3
		cookies []string
	)

	defer func() {
		for _, value := range cookies {
			resetMarker(value)
		}
	}()

	var (
		proxies    = ctx.GetString("proxies")
		completion = common.GetGinCompletion(ctx)
		matchers   = common.GetGinMatchers(ctx)
	)

	completion.Model = completion.Model[4:]

	messages := make([]string, 0)
	if completion.System != "" {
		messages = append(messages, completion.System)
	}

	for _, message := range completion.Messages {
		messages = append(messages, message.GetString("content"))
	}

	if youRollContainer.Len() == 0 {
		response.Error(ctx, -1, "empty cookies")
		return
	}

label:
	retry--
	cookie, err := youRollContainer.Poll()
	if err != nil {
		logger.Error(err)
		response.Error(ctx, -1, err)
		return
	}

	cookies = append(cookies, cookie)
	chat := you.New(cookie, completion.Model, proxies)
	chat.LimitWithE(true)
	chat.Client(plugin.HTTPClient)

	if err = tryCloudFlare(); err != nil {
		response.Error(ctx, -1, err)
		return
	}

	chat.CloudFlare(clearance, userAgent, lang)
	fileMessage := strings.Join(messages, "\n\n")
	ch, err := chat.Reply(common.GetGinContext(ctx), nil, fileMessage, notice)
	if err != nil {
		logger.Error(err)
		var se emit.Error
		code := -1
		if errors.As(err, &se) && se.Code > 400 {
			_ = youRollContainer.SetMarker(cookie, 2)
			// 403 重定向？？？
			if se.Code == 403 {
				code = 429
				cleanCf()
			}
		}

		if strings.Contains(err.Error(), "ZERO QUOTA") {
			_ = youRollContainer.SetMarker(cookie, 2)
			code = 429
		}

		if retry > 0 {
			goto label
		}
		response.Error(ctx, code, err)
		return
	}

	var cancel chan error
	cancel, matchers = joinMatchers(ctx, matchers)
	content := waitMessageResponse(ctx, ch, matchers, cancel)
	if content == "" && response.NotResponse(ctx) {
		response.Error(ctx, -1, "EMPTY RESPONSE")
	}
}

func cleanCf() {
	mu.Lock()
	clearance = ""
	mu.Unlock()
}

func resetMarker(cookie string) {
	marker, e := youRollContainer.GetMarker(cookie)
	if e != nil {
		logger.Error(e)
		return
	}

	if marker != 1 {
		return
	}

	e = youRollContainer.SetMarker(cookie, 0)
	if e != nil {
		logger.Error(e)
	}
}

func tryCloudFlare() error {
	if clearance == "" {
		logger.Info("trying cloudflare ...")

		mu.Lock()
		defer mu.Unlock()
		if clearance != "" {
			return nil
		}

		port := pkg.Config.GetString("you.helper")
		r, err := emit.ClientBuilder(plugin.HTTPClient).
			GET("http://127.0.0.1:"+port+"/clearance").
			DoC(emit.Status(http.StatusOK), emit.IsJSON)
		if err != nil {
			logger.Error(err)
			if emit.IsJSON(r) == nil {
				logger.Error(emit.TextResponse(r))
			}
			return err
		}

		defer r.Body.Close()
		obj, err := emit.ToMap(r)
		if err != nil {
			logger.Error(err)
			return err
		}

		data := obj["data"].(map[string]interface{})
		clearance = data["cookie"].(string)
		userAgent = data["userAgent"].(string)
		lang = data["lang"].(string)
	}
	return nil
}

func joinMatchers(ctx *gin.Context, matchers []common.Matcher) (chan error, []common.Matcher) {
	// 自定义标记块中断
	keyv, ok := common.GetGinValue[pkg.Keyv[string]](ctx, vars.GinCharSequences)
	if ok {
		if user := keyv.GetString("user"); user == "" {
			keyv.Set("user", "\n\nHuman:")
			ctx.Set(vars.GinCharSequences, keyv)
		}
	}

	cancel, matcher := common.NewCancelMatcher(ctx)
	matchers = append(matchers, matcher...)
	return cancel, matchers
}

func Condition(cookie string) bool {
	marker, err := youRollContainer.GetMarker(cookie)
	if err != nil {
		logger.Error(err)
		return false
	}

	if marker != 0 {
		return false
	}

	//return true
	chat := you.New(cookie, you.CLAUDE_2, vars.Proxies)
	chat.Client(plugin.HTTPClient)
	chat.CloudFlare(clearance, userAgent, lang)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	// 检查可用次数
	count, err := chat.State(ctx)
	if err != nil {
		var se emit.Error
		if errors.As(err, &se) {
			if se.Code == 403 {
				cleanCf()
				_ = tryCloudFlare()
			}
			if se.Code == 401 { // cookie 失效？？？
				_ = youRollContainer.SetMarker(cookie, 2)
			}
		}
		logger.Error(err)
		return false
	}

	if count <= 0 {
		_ = youRollContainer.SetMarker(cookie, 2)
		return false
	}

	return true
}
