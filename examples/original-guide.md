<!-- Historical source; not active instructions. Gist revision: 8bee3c44536d681ad463dbe817808d6c60753e1a -->

# Tyson Gern's Writing Style Guide

A portable reference for drafting and reviewing prose in my voice.

## Goal

Writing should be succinct, professional, and almost academic. It should make defensible claims, avoid hype, and help the reader understand something concrete. The aim is precise, useful prose that sounds recognizably mine.

## Hard Rules

Follow these strictly. Flag any violation.
- Omit needless words.
- Keep the tone succinct and professional, almost academic.
- Avoid grandiose, inflated, or hard-to-verify claims.
- Avoid self-referential phrases like "this paper", "this talk", or "this meetup". Use a direct noun instead.
- Avoid business speak, marketing language, and buzzwords. Words like "drive", "leverage", "unlock", "robust", "seamless", and "corpus" do not belong.
- Avoid negative parallelism, such as "It's not A. It's B."
- Do not mix metaphors. Avoid stale metaphors borrowed from generic tech writing.
- Give each subject only actions it performs and properties it has. When wording assigns human needs, intentions, perceptions, or decisions to a thing, name the actual actor and action, or state the concrete operation, property, condition, or dependency.
- Give software and mathematics the verbs of their function, never verbs of mind. A system returns, ranks, scores, serves, catches; it does not notice, want, wander, forgive, or punish. If the verb ascribes intent, perception, or emotion to a thing, replace it with the operation it performs. API vocabulary is exempt when it names the actual call.
- Check every verb in the sentence, not only the main one. Causative constructions launder a borrowed verb into the object: in "unit vectors make the three measures agree", the violation is "agree", hidden behind "make".
- Do not promote a condition into an agent. When a fact holds under a condition, state the condition in a when, for, or on phrase rather than making it the subject of "make", "let", "give", "cause", or "ensure". Write "for unit vectors, the three measures produce the same ranking", never "unit vectors make them agree". Test: if the subject can be rewritten as a when-clause without changing the claim, it was a condition, not an actor.
- Give data properties, never actions. A word weighs zero, one match outscores another, one term dominates the sum. A word does not decide or determine the ranking: ranking is the scoring function's act, so attribute the act to the system or restate it as a property of the thing. A metric names a quantity, not its arithmetic: precision does not divide the overlap by anything; it is the overlap divided by what came back. Test: does the verb name a property or comparison of the subject, or an outcome in the world? A people-verb survives only as the field's standard name for a relation (matrices commute) with no plain substitute; "agree" between quantities fails because "match" is available.
- Describe the thing itself, not the conceptual machinery used to explain it. Use concrete subjects and direct claims: say what happens, what someone does, or what is true. Avoid making abstractions perform analytical actions. Write "A match ends in a home win, a draw, or an away win", not "The score space partitions into three outcomes". If the claim is that we group, divide, or classify something, name the actor or state the categories directly. Apply this rule to headlines as well as sentences. Retain technical abstractions when their precision matters. Test: what is the claim about the underlying thing? State that claim directly.
- Do not call something simple or easy unless it obviously is. Most things are not.
- Prefer plain English over jargon, foreign phrases, or long words.
- Prefer active voice over passive voice.
- Avoid qualifiers like "rather", "very", "little", and "pretty". They weaken the sentence.
- Do not affect a breezy manner. The voice should stay measured.
- Never use emdashes. Use periods, commas, parentheses, or colons instead.
- Do not fabricate business context. If a detail isn't grounded in something I've told you, leave it out.
- Do not end by restating the thesis in louder language.
- The opening sentence of an article or post should say something. Avoid inane setups like "everyone knows that programming is difficult."

## Strong Preferences

Writing should follow these directions, though exceptions exist.
- State the claim early. The first paragraph should name the problem, tradeoff, or thesis.
- Use precise nouns and verbs in place of abstractions.
- Define the disagreement or tradeoff before recommending an approach.
- Use examples to clarify a point, not to decorate it.
- Prefer measured recommendations over universal prescriptions.
- Sound like a competent practitioner explaining what they have seen in practice.
- Organize sections so each one advances the argument.
- End by reinforcing the recommendation, not by escalating the rhetoric.
- Put statements in positive form. Say what is, not what isn't. "He usually arrived late" beats "He did not often arrive on time."
- Use parallel construction for coordinate ideas. When two or more ideas share a structural role, give them the same grammatical shape.
- Place the emphatic word at the end of the sentence. The end is the position of stress.
- Do not overwrite. Avoid ornament the argument does not need.
- Place yourself in the background. Let the subject lead.

## Observed Habits

Patterns from my published writing worth emulating.
- Start with a concrete problem or recurring practice.
- Name why people do the common thing before arguing against it.
- Shift quickly from description to recommendation.
- Frame decisions in terms of cost, confidence, complexity, maintainability, or clarity.
- Use comparison to sharpen an argument, but keep the comparison literal.
- Prefer direct recommendations such as "start with", "avoid", "use", "consider", and "treat".
- Make room for exceptions and tradeoffs.
- Use section headings to keep the argument linear.

## Short-Form Habits

For short pieces, social posts, and edits, these specific nudges apply.
- Prefer indefinite articles ("a", "an") when introducing things, rather than definite ("the").
- Prefer comparative claims ("faster than", "smaller than") over binary claims ("fast", "small").
- Avoid second person ("you"). Use first person or a named subject instead.
- Prefer "then" over "and" when describing a sequence.
- Lead with the action. "Run the migration" beats "the migration runs".
- When mentioning tools, speak in a personal voice ("I use X") rather than recommending universally.

## Rewrites

When tightening a draft, stay within the author's claim. Tighten and clarify. Do not add new arguments, examples, or context. A rewrite that changes what the author is saying is not a rewrite.

## Examples From My Published Writing

These short excerpts show the shape of the prose, the level of confidence, and the kinds of claims I make.

### 1. Replace taxonomy fights with a clearer distinction

> Rather than arguing over which labels to use and how to define them, I've found it more helpful to use one of two adjectives to label each test: slow or fast.

It starts from a common problem, moves quickly to a recommendation, and stays modest and practical.

### 2. Recommend through cost and consequence

> The choice of test labels is an important influence on the makeup of a test suite.
>
> When we get this wrong, we end up with a test suite that either doesn't provide accurate coverage or provides coverage at an unacceptable cost.

The first sentence names the subject plainly. The second explains the consequence without exaggeration. The argument is framed in terms of cost and coverage, not ideology.

### 3. Argue against a common practice without caricature

> Developers use CSS frameworks like Material UI, Bootstrap, or Pico to reduce boilerplate, increase quality, and drive consistency.
>
> However, these gains are hard to maintain as an application's codebase matures.

It states why the opposing approach is attractive, then turns to the limitation immediately. The criticism is measured rather than dismissive.

### 4. Make the recommendation explicit

> Instead of using a CSS framework, I recommend developers write their own custom CSS.

The recommendation is unmistakable. It does not hide behind hedging or scene-setting. The sentence is strong without sounding promotional.

### 5. Explain tradeoffs in plain language

> These frameworks are simpler than SPAs but retain the flexibility of an SPA by allowing developers to run arbitrary code in the browser.
>
> They come with a bit of complexity, as much of the application's code must be able to run both in the browser and on the backend.

It compares options on specific dimensions and avoids false binaries. The prose stays technical and literal.

### 6. Moderate the claim and leave room for reality

> As long as we're using fast, reliable collaborators, testing with real collaborators doesn't negatively impact the speed and reliability of our tests.
>
> When this is not the case, test doubles are a great way to increase a test's speed and reliability while sacrificing a bit of confidence, clarity, and flexibility.

It recommends a default, then names the exception. The tradeoff is explicit. There are no universal rules.

## Reference Principles

The rules above derive from three sources I treat as authoritative.

### From Vonnegut, *How to Write with Style*

- Find a subject you care about.
- Do not ramble.
- Keep it simple.
- Have the guts to cut.
- Sound like yourself.
- Say what you mean.
- Pity the reader.

### From Orwell, *Politics and the English Language*

1. Never use a metaphor, simile, or other figure of speech which you are used to seeing in print.
2. Never use a long word where a short one will do.
3. If it is possible to cut a word out, always cut it out.
4. Never use the passive where you can use the active.
5. Never use a foreign phrase, a scientific word, or a jargon word if you can think of an everyday English equivalent.
6. Break any of these rules sooner than say anything outright barbarous.

### From Strunk and White, *The Elements of Style*

- Put statements in positive form.
- Use definite, specific, concrete language.
- Use parallel construction for coordinate ideas.
- Place the emphatic word of a sentence at the end.
- Avoid qualifiers like "rather", "very", "little", and "pretty".
- Do not overwrite.
- Do not affect a breezy manner.
- Place yourself in the background.

## Review Process

When reviewing a draft against this guide, structure feedback in this order.
1. Weak openings, vague wording, and inflated claims.
2. Structural problems that obscure the argument.
3. Sentence-level clarity and word choice.
4. Tone drift.
5. Minor polish.

For each finding, cite the source principle. Good citations look like:
- "Hard rule: avoid phrases like 'this talk'."
- "Strong preference: state the claim early."
- "Orwell: if it is possible to cut a word, cut it."
- "Vonnegut: do not ramble."
- "Observed habit: acknowledge why the common approach is attractive before criticizing it."
If a finding cannot be tied to a principle in this guide, leave it out.

## Published Samples

For longer-form reference, these articles represent my current style.
- [A Simpler Testing Pyramid: Getting the Most out of Your Tests](https://www.infoq.com/articles/testing-pyramid-slow-fast-tests/)
- [The Unit in Unit Testing](https://www.infoq.com/articles/unit-testing-approach/)
- [You Don't Need a CSS Framework](https://www.infoq.com/articles/no-need-css-framework/)
- [Streaming HTML: Asynchronous DOM Updates without JavaScript](https://www.infoq.com/articles/html-streaming-dom-updates-without-javascript/)
