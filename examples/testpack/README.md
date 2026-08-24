# DataCraft Minecraft test pack

Install the generated `build/testpack` folder in a world's `datapacks` folder,
then enter or reload the world. The pack runs automatically from its load
function and prints 94 diagnostic results in chat. Mixed-list iteration also
prints its individual runtime value/type lines.

Every line includes the actual and expected result. Report any line that is
missing or whose `got` value differs from `expected`.

To run it again without reloading the world:

```mcfunction
/function testpack:run_tests
```
