import { expect, test, type Request, type Route } from "@playwright/test";

test("auto-saves prep answers and restores after refresh", async ({ page }) => {
  let saveCalls = 0;
  let answers = [{ question_id: 1, answer: "" }];
  let savedAt = "2026-03-27T10:05:00Z";

  await page.route("**/api/prep/meta", async (route: Route) => {
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({
        data: {
          enabled: true,
          default_question_count: 2,
          supported_scopes: ["topics"],
        },
      }),
    });
  });

  await page.route("**/api/prep/index/status", async (route: Route) => {
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({
        data: {
          embedding_provider: "stub",
          embedding_model: "stub-v1",
          document_count: 0,
          chunk_count: 0,
          last_indexed_at: "",
          last_index_status: "",
        },
      }),
    });
  });

  await page.route("**/api/prep/leads/lead_test/context-preview", async (route: Route) => {
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({
        data: {
          lead_id: "lead_test",
          company: "OpenAI",
          position: "Agent Engineer",
          has_resume: true,
          topic_keys: ["rag"],
          sources: [],
        },
      }),
    });
  });

  await page.route("**/api/prep/sessions/stream", async (route: Route, request: Request) => {
    if (request.method() !== "POST") {
      await route.continue();
      return;
    }

    const streamBody = [
      "event: started",
      'data: {"lead_id":"lead_test"}',
      "",
      "event: completed",
      `data: ${JSON.stringify({
        session: {
          id: "prep_01",
          lead_id: "lead_test",
          company: "OpenAI",
          position: "Agent Engineer",
          status: "draft",
          config: {
            topic_keys: ["rag"],
            question_count: 2,
            include_resume: true,
            include_lead_docs: true,
          },
          sources: [],
          questions: [
            { id: 1, type: "open", content: "What is RAG?", expected_points: [], context_sources: [] },
            { id: 2, type: "open", content: "How does retrieval work?", expected_points: [], context_sources: [] },
          ],
          answers,
          reference_answers: {},
          created_at: "2026-03-27T10:00:00Z",
          updated_at: savedAt,
        },
      })}`,
      "",
      "",
    ].join("\n");

    await route.fulfill({
      status: 200,
      contentType: "text/event-stream",
      body: streamBody,
    });
  });

  await page.route("**/api/prep/sessions/prep_01", async (route: Route, request: Request) => {
    if (request.method() !== "GET") {
      await route.continue();
      return;
    }

    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({
        data: {
          id: "prep_01",
          lead_id: "lead_test",
          company: "OpenAI",
          position: "Agent Engineer",
          status: "draft",
          config: {
            topic_keys: ["rag"],
            question_count: 2,
            include_resume: true,
            include_lead_docs: true,
          },
          sources: [],
          questions: [
            { id: 1, type: "open", content: "What is RAG?", expected_points: [], context_sources: [] },
            { id: 2, type: "open", content: "How does retrieval work?", expected_points: [], context_sources: [] },
          ],
          answers,
          reference_answers: {},
          created_at: "2026-03-27T10:00:00Z",
          updated_at: savedAt,
        },
      }),
    });
  });

  await page.route("**/api/prep/sessions/prep_01/draft-answers", async (route: Route, request: Request) => {
    if (request.method() !== "PUT") {
      await route.continue();
      return;
    }

    saveCalls += 1;
    const payload = request.postDataJSON() as { answers?: Array<{ question_id: number; answer: string }> };
    answers = Array.isArray(payload.answers) ? payload.answers : [];
    savedAt = "2026-03-27T10:06:00Z";

    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({
        data: {
          session_id: "prep_01",
          saved_at: savedAt,
          answers_count: answers.length,
        },
      }),
    });
  });

  await page.goto("/prep?lead_id=lead_test");
  await page.getByRole("tab", { name: "练习" }).click();
  await page.getByRole("button", { name: "生成题目" }).click();

  const answerInput = page.getByTestId("prep-answer-input-1");
  await expect(answerInput).toBeVisible();
  await answerInput.fill("RAG stands for Retrieval-Augmented Generation.");

  await expect.poll(() => saveCalls, { timeout: 6000 }).toBeGreaterThan(0);

  await page.reload();
  await page.getByRole("tab", { name: "练习" }).click();

  await expect(page.getByTestId("prep-answer-input-1")).toHaveValue("RAG stands for Retrieval-Augmented Generation.");
});
