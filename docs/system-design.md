# Timeline system design

Timeline is a separate, reusable system for humans and agents to record and follow items over time. It distinguishes information recorded after something happened, plans or expectations recorded beforehand, and observations recorded as something happens.

This document describes what Timeline should do. [Architecture](architecture.md) describes how to build it.

## Items and types

Users can define item types for their own use case. Each type defines:

- its name and the folder where its items belong
- the information each item must contain
- any allowed statuses
- the evidence or source references it requires
- its entry timing rule

Each item is a readable Markdown document in the folder for its type. It contains metadata for tracking and organizing it, along with its written content. Humans and agents can add, read, and update items.

News, predictions, and tasks are possible types, not a fixed list built into Timeline. The system should work for other uses without needing investment concepts, a World Model, or a particular agent.

## Two times for every item

Each item records:

- **Event time:** when something happened, is expected to happen, or is due.
- **Entry time:** when the item first entered the system.

For example, a report entered on Friday about something that happened on Monday has Monday as its event time and Friday as its entry time. A plan entered on Monday for Friday has those times the other way around.

Keeping both times lets readers distinguish what was known in advance from what was recorded afterward.

## Entry timing rules

An entry timing rule describes how an item's event time relates to its entry time. It belongs to the item type.

| Rule | Meaning | Typical information needed |
| --- | --- | --- |
| Past | The event happened before the item was added | Evidence of what happened and when, and its source |
| Future | The item was added before the expected event or planned work | Reasons or plans, and what will be collected later to check the result |
| Real-time | The item was added as the event was happening | A current observation, its source, and its freshness |

A real-time type defines how much reporting delay it accepts. An old observation must not appear to be live just because it was entered now.

The system checks an item's timing against its type's rule. It also checks that the required information is present. It does not decide whether a claim is true or whether the evidence is convincing; those judgments belong to the people, agents, or applications using it.

## Time passing and later updates

The entry timing rule describes the item at first entry. It does not change when the clock moves forward. An item entered under the Future rule remains something recorded in advance even after its event date passes.

Timeline should show both the original entry timing and where the event sits relative to now. Users can find past events, upcoming events, and items whose dates have passed but whose configured status is still unfinished.

Later updates can add results, evidence, notes, or status changes allowed by the type. They preserve the original entry time and earlier content. A change to a type's definition must not silently change the meaning or validity of items already recorded under it.

## Reading and finding items

Users should be able to:

- read items in event-time order
- search or filter by event time, entry time, type, entry timing rule, and configured metadata such as owner or status
- open the document behind each result
- see an item's history and who changed it
- find records with missing, invalid, or conflicting information

A view should distinguish current validated information from changes that have not yet been accepted or indexed. Missing or invalid records must not disappear silently from a supposedly complete result.

## Links and history

Items can link to other items, sections, external documents, or sources. Ordinary Markdown links are enough. Timeline preserves those links without needing to understand what each connection means.

Every item keeps its identity when it moves. Corrections and removals remain visible in history. Applications should be able to recover an earlier recorded version rather than mistaking today's document for what was known then.

## Independence

Timeline remains useful without a running agent. Changing the software used to read and write it does not lose items or their history. Multiple writers must not silently overwrite one another's changes.

The first scope is item definitions, documents, timing checks, history, and queries. Domain-specific work such as judging predictions, calculating learning accuracy, executing tasks, or maintaining a World Model belongs to the application using Timeline. Automatic link repair is deferred.

## Database independence

The database is a replaceable, rebuildable index; Markdown, configuration, and accepted Git history remain authoritative. SQLite is the initial local backend. Users can configure another supported database without changing item documents or Timeline behavior. New backend implementations must preserve query and recovery semantics. See [database requirements](requirements/TL-DATABASE.md) for selection, target switching, and verification.
