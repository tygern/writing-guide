# Sentence editing examples

These examples are constructed editing demonstrations, not quotations from Tyson's publications. More than one rewrite can satisfy a rule. The context must support every added detail.

| Original | Treatment | Rule and reason |
| --- | --- | --- |
| The index needs updating. | Update the index. | `subject-predicate`: express the action directly without inventing an owner. |
| The evaluation needs rerunning when the model changes. | Rerun the evaluation when the model changes. | `subject-predicate`, `preserve-meaning`: preserve the trigger. |
| The model wants a fixed-length input. | The model accepts only fixed-length inputs. | `subject-predicate`: use this rewrite when fixed length is an actual constraint. If that is unclear, ask or flag the ambiguity. |
| The service requires an API key. | Keep. | `subject-predicate`: a concrete dependency. |
| The function returns a promise. | Keep. | `subject-predicate`: an operation using API vocabulary. |
| Unit vectors make the three measures agree. | For unit vectors, the three measures produce the same ranking. | `subject-predicate`, `preserve-meaning`: use this only when ranking equivalence is the intended claim. |
| Precision divides relevant returned results by all returned results. | Precision is the number of relevant results returned divided by the total number returned. | `subject-predicate`: define the quantity. |
| Retries increase request volume. | Keep. | `subject-predicate`: a causal claim does not imply intent. |
| There is little evidence that this change improves recall. | Keep. | `measured-language`: the qualification carries meaning. |
| The request may time out under load. | Keep. | `preserve-meaning`: removing “may” changes possibility to certainty. |
| The records were deleted at noon. | Keep if the actor is unknown. | `preserve-meaning`, `direct-prose`: do not invent an operator. |
| It's not about embeddings. It's about retrieval quality. | Evaluate retrieval quality. | `direct-prose`: appropriate when the intended claim is a recommendation to evaluate. Preserve additional substantive distinctions if present. |
| This talk explores the importance of evaluating search. | Evaluate search on representative queries. | `concrete-claims`: use only when representative queries are part of the supplied argument. |

## Check the claim before choosing a rewrite

“The model wants a fixed-length input” might refer to an interface constraint, a performance preference, or merely a poorly supported assertion. A rewrite must preserve the actual claim. Changing “wants” to “requires” without checking the meaning can strengthen it without evidence.

“Faster” can be precise when the comparison is supplied. Replacing “fast” with “faster” does not provide the missing comparison or measurement.
