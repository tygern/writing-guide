# Annotated published excerpts

These quotations are preserved verbatim from the original Gist. They are author-supplied excerpts, not newly generated writing. The annotations describe specific qualities to retain and conflicts with the current guide. Current rules take precedence when drafting new prose. The archive retains the original publication links; individual excerpt-to-article mappings have not been independently verified.

## 1. Replace taxonomy fights with a clearer distinction

> Rather than arguing over which labels to use and how to define them, I've found it more helpful to use one of two adjectives to label each test: slow or fast.

The sentence moves from a terminology dispute to a usable distinction. The first-person claim is supported by the original author. “Rather than” expresses a comparison; it is not an empty intensifier. Apply `concrete-claims` and `personal-voice`.

## 2. Recommend through cost and consequence

> The choice of test labels is an important influence on the makeup of a test suite.
>
> When we get this wrong, we end up with a test suite that either doesn't provide accurate coverage or provides coverage at an unacceptable cost.

The second paragraph explains the consequences of a choice: inaccurate coverage or excessive cost. Preserve both outcomes when editing. Apply `argument-structure` and `preserve-meaning`.

## 3. Argue against a common practice without caricature

> Developers use CSS frameworks like Material UI, Bootstrap, or Pico to reduce boilerplate, increase quality, and drive consistency.
>
> However, these gains are hard to maintain as an application's codebase matures.

The opening gives reasons practitioners choose frameworks before describing the maintenance problem. This demonstrates `argument-structure`. The phrase “drive consistency” conflicts with current `measured-language`; preserve the quotation but do not imitate that phrase in new prose.

## 4. Make the recommendation explicit

> Instead of using a CSS framework, I recommend developers write their own custom CSS.

The recommendation names the person making it and the action proposed. Apply `direct-prose` and `personal-voice`. This historical position does not establish a recommendation for every new task.

## 5. Explain tradeoffs in plain language

> These frameworks are simpler than SPAs but retain the flexibility of an SPA by allowing developers to run arbitrary code in the browser.
>
> They come with a bit of complexity, as much of the application's code must be able to run both in the browser and on the backend.

The passage connects a comparison to a specific capability and its implementation cost. Apply `argument-structure`. “Simpler” has an explicit comparison with SPAs. “A bit of complexity” is imprecise under current `measured-language`; preserve the quotation while preferring a concrete account of the cost in new prose.

## 6. Moderate the claim and leave room for reality

> As long as we're using fast, reliable collaborators, testing with real collaborators doesn't negatively impact the speed and reliability of our tests.
>
> When this is not the case, test doubles are a great way to increase a test's speed and reliability while sacrificing a bit of confidence, clarity, and flexibility.

The recommendation is conditional on collaborator characteristics, and the alternative has an explicit tradeoff. Apply `preserve-meaning` and `measured-language`. “Great way” is evaluative wording that should not be copied without a basis; negation in the original expresses an actual claim and is not automatically a violation.
